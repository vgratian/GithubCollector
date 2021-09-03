package github

import (
	"context"
	"encoding/json"
	"goharvest2/cmd/poller/collector"
	//"goharvest2/cmd/poller/registrar"
	"goharvest2/cmd/poller/plugin"
	"goharvest2/pkg/errors"
	"goharvest2/pkg/matrix"
	"goharvest2/pkg/tree/node"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
	"strings"
)

func (Github) HarvestModule() plugin.ModuleInfo {
	return plugin.ModuleInfo{
		ID:  "harvest.collector.github",
		New: func() plugin.Module { return new(Github) },
	}
}

// Interface guards
var (
	_ collector.Collector = (*Github)(nil)
)

func init() {
	//registrar.RegisterCollector("Github", func() collector.Collector { return new(Github) })
	plugin.RegisterModule(Github{})
}

// Github collector
type Github struct {
	*collector.AbstractCollector
	client         *http.Client
	urlPrefix      string
	repoName       string
	repoPath       string
	elements       map[string]*Element
	matrices       map[string]*matrix.Matrix
	sizeMetric		matrix.Metric
	countMetric     matrix.Metric
}

func (me *Github) Init(a *collector.AbstractCollector) error {

	var (
		err                      error
		token, interval          string
		obj, counters, intervals *node.Node
		ctx                      context.Context
		src                      oauth2.TokenSource
		elem                     *Element
	)

	me.AbstractCollector = a
	me.Logger.Debug().Msg("initializing Github collector!")

	// Invoke generic initializer
	if err = collector.Init(me); err != nil {
		return err
	}

	// addr should contain the repo name, e.g. "NetApp/harvest"
	if me.repoPath = me.Params.GetChildContentS("addr"); me.repoPath == "" {
		return errors.New(errors.MISSING_PARAM, "addr")
	}

	me.repoPath = strings.TrimSuffix(me.repoPath, "/")

    if s := strings.Split(me.repoPath, "/"); len(s) < 2 {
        me.Logger.Warn().Msgf("expected should have [https:github.com/]OWNER/NAME format", me.repoPath)
        return errors.New(errors.INVALID_PARAM, "addr")
    } else {
        me.repoName = s[len(s)-1]
		me.repoPath = s[len(s)-2] + "/" + me.repoName
    }
	me.urlPrefix = "https://api.github.com/repos/" + me.repoPath
	me.Logger.Debug().Msgf("target [%s]: will query APIs in [%s]", me.repoPath, me.urlPrefix)

	// construct HTTP client
	if token = me.Params.GetChildContentS("password"); token != "" {
		src = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		me.Logger.Info().Msgf("will use OAuth2 token [%s]", strings.Repeat("*", len(token)))
	} else {
		me.Logger.Info().Msg("will use no Github authentication")
	}
	ctx = context.Background()
	me.client = oauth2.NewClient(ctx, src)

	// initialize one matrix for each API endpoint

    me.elements = make(map[string]*Element)
    me.matrices = make(map[string]*matrix.Matrix)

	if counters = me.Params.GetChildS("counters"); counters == nil {
		return errors.New(errors.MISSING_PARAM, "counters")
	}

	// user-defined intervals for each API poll
	intervals = me.Params.GetChildS("intervals")

	for _, obj = range counters.GetChildren() {

		elem = ParseElementTree(obj)

		me.elements[elem.Name] = elem
		me.matrices[elem.Name] = matrix.New("Github."+elem.Name, elem.DisplayName)
		me.matrices[elem.Name].SetGlobalLabel("repo", me.repoPath)

		if elem.HasNestedKeys {
			me.Logger.Debug().Msgf("[%s] (%s) will use parsed instance keys", elem.Name, elem.DisplayName)
		} else {
			me.Logger.Debug().Msgf("[%s] (%s) will use single instance (repo)", elem.Name, elem.DisplayName)
			if _, err = me.matrices[elem.Name].NewInstance("repo"); err != nil {
				return err
			}
		}

		if err = me.createMetrics(elem.Name, elem); err != nil {
			return err
		}

		// little hack to create a schedule for this API

		if intervals == nil {
			interval = "15m"
		} else if interval = intervals.GetChildContentS(elem.DisplayName); interval == "" {
			interval = "15m"
		}

		taskName := (*elem).Name
		err = me.Schedule.NewTaskString(
			taskName,
			interval,
			func() (*matrix.Matrix, error) {
				return me.PollThis(taskName)
			},
			true,
			"",
		)
		if err != nil {
			return err
		}
		me.Logger.Info().Msgf("(%s) set schedule to: %s", elem.Name, interval)
	}

	// we will use me.Matrix to record size and line count in files
	if me.sizeMetric, err = me.Matrix.NewMetricUint64("size_bytes"); err != nil {
		return err
	}

	if me.countMetric, err = me.Matrix.NewMetricUint64("size_lines"); err != nil {
		return err
	}

	return nil
}

// keep dummy function, just because we have "data" task
func (me *Github) PollData() (*matrix.Matrix, error) {
	return nil, nil
}

func (me *Github) PollThis(key string) (*matrix.Matrix, error) {
	var (
		code    int
		data    []byte
		dataMap map[string]json.RawMessage
        dataMapSlice []map[string]json.RawMessage
		err     error
	)

    me.Logger.Info().Msgf("[%s] (%s) Starting poll", key, me.elements[key].DisplayName)

	if code, data, err = me.DoRequest(key); err != nil {
		return nil, err
	} else if code != 200 {
		me.Logger.Warn().Msgf("API Response: %d", code)
		return nil, nil
	}

    if me.elements[key].HasNestedKeys {
        me.Logger.Debug().Msgf("(%s) unmarshalling json into slice", key)
        if err = json.Unmarshal(data, &dataMapSlice); err != nil {
            return nil, err
        }
    } else {
		me.Logger.Debug().Msgf("(%s) unmarshalling json into map", key)
		if err = json.Unmarshal(data, &dataMap); err != nil {
			return nil, err
		}
        err = me.extractJsonData(key, me.elements[key], dataMap, "", nil)
		return me.matrices[key], err
	}

    for _, m := range dataMapSlice {
        if err = me.extractJsonData(key, me.elements[key], m, "", nil); err != nil {
            return nil, err
        }
    }

	return me.matrices[key], err
}

func (me *Github) DoRequest(reqSuffix string) (int, []byte, error) {
	var (
		data    []byte
		resp    *http.Response
		err     error
		fullUrl string
	)

	fullUrl = me.urlPrefix + reqSuffix

	me.Logger.Debug().Msgf(" ~> issuing request [%s]", fullUrl)

	if resp, err = me.client.Get(fullUrl); err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()

	data, err = ioutil.ReadAll(resp.Body)

	return resp.StatusCode, data, err
}


// Create metrics for elements that are a metric
func (me *Github) createMetrics(k string, e *Element) error {
    var (
        metric matrix.Metric
        err error
    )
    me.Logger.Debug().Msgf("(%s) handling [%s]", k, e.String())

    if e.IsMetric {
		me.Logger.Debug().Msgf("(%s) => metric    [%s] (%s)", k, e.Name, e.DisplayName)
        if metric, err = me.matrices[k].NewMetricInt64(e.Name); err != nil {
            return err
        }
        metric.SetName(e.DisplayName)
    }

    if e.IsNested {
        for _, c := range e.Elements {
            if err = me.createMetrics(k, c); err != nil {
                return err
            }
        }
    }

    return nil
}
