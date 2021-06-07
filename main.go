package github

import (
	"encoding/json"
	"goharvest2/cmd/poller/collector"
	"goharvest2/cmd/poller/registrar"
	"goharvest2/pkg/errors"
	"goharvest2/pkg/matrix"
	"goharvest2/pkg/tree/node"
	"io/ioutil"
	"net/http"
	"strings"
)

// Github collector
type Github struct {
	*collector.AbstractCollector
	urlPrefix string
}

// Interface guards
var (
	_ collector.Collector = (*Github)(nil)
)

func init() {
    registrar.RegisterCollector("Github", func() collector.Collector { return new(Github) })
}

type repoData struct {
	Size       uint64 `json:"size"`
	Stars      uint64 `json:"stargazers_count"`
	Forks      uint64 `json:"forks_count"`
	OpenIssues uint64 `json:"open_issues_count"`
}

var repositoryMetrics = []string{"size", "stargazers_count", "forks_count", "open_issues_count"}

func (me *Github) Init(a *collector.AbstractCollector) error {

	var (
		err      error
		addr     string
		repos    *node.Node
		instance *matrix.Instance
	)

	me.AbstractCollector = a
	me.Logger.Debug().Msg("initializing Github collector!")

	// Invoke generic initializer
	if err = collector.Init(me); err != nil {
		return err
	}

	// construct REST API url of target repository
	if addr = me.Params.GetChildContentS("addr"); addr == "" {
		return errors.New(errors.MISSING_PARAM, "addr")
	}

	me.urlPrefix = strings.Replace(addr, "github.com", "api.github.com/repos", 1)
	me.urlPrefix = strings.TrimSuffix(me.urlPrefix, "/")
	me.Logger.Debug().Msgf("using API url prefix [%s]", me.urlPrefix)

	// construct instance cache
	if repos = me.Params.GetChildS("repos"); repos == nil {
		return errors.New(errors.MISSING_PARAM, "repos")
	}
	for _, repo := range repos.GetAllChildContentS() {
		if instance, err = me.Matrix.NewInstance(strings.TrimSuffix(repo, "/")); err != nil {
			return err
		}
		instance.SetLabel("repo", repo)
	}

	// construct metric cache
	for _, metricName := range repositoryMetrics {
		if _, err = me.Matrix.NewMetricUint64(metricName); err != nil {
			return err
		}
	}

	me.Logger.Debug().Msgf("initialized cache with %d instances and %d metrics", len(me.Matrix.GetInstances()), len(me.Matrix.GetMetrics()))
	return nil

}

func (me *Github) DoRequest(requestUrl, repoName string) (int, []byte, error) {
	var (
		data    []byte
		resp    *http.Response
		err     error
		fullUrl string
	)

	fullUrl = me.urlPrefix + "/" + repoName + requestUrl

	me.Logger.Debug().Msgf("issuing request [%s]", fullUrl)

	if resp, err = http.Get(fullUrl); err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()

	data, err = ioutil.ReadAll(resp.Body)

	return resp.StatusCode, data, err
}

func (me *Github) PollData() (*matrix.Matrix, error) {
	var (
		rawData  []byte
		dataMap  map[string]json.RawMessage
		value    uint64
		respCode int
		err      error
	)

	me.Matrix.Reset()

	for key, instance := range me.Matrix.GetInstances() {

		// query basic repo stats
		if respCode, rawData, err = me.DoRequest("", key); err != nil {
			return nil, err
		}

		dataMap = make(map[string]json.RawMessage)
		if err = json.Unmarshal(rawData, &dataMap); err != nil {
			return nil, err
		}

		if respCode != 200 {
			me.Logger.Warn().Msgf("http response: [%d] %s", respCode, string(dataMap["message"]))
			continue
		}

		for name, metric := range me.Matrix.GetMetrics() {

			if strings.HasPrefix(name, "languages.") {
				continue
			}

			if err = json.Unmarshal(dataMap[name], &value); err != nil {
				me.Logger.Error().Stack().Err(err).Msgf("parse (%s) value [%s]", name, dataMap[name])
			} else if err = metric.SetValueUint64(instance, value); err != nil {
				me.Logger.Error().Stack().Err(err).Msgf("set (%s) value [%d]", name, value)
			}
		}

		// query list of languages
		if respCode, rawData, err = me.DoRequest("/languages", key); err != nil {
			return nil, err
		}

		dataMap = make(map[string]json.RawMessage)
		if err = json.Unmarshal(rawData, &dataMap); err != nil {
			return nil, err
		}

		if respCode != 200 {
			me.Logger.Warn().Msgf("http response: [%d] %s", respCode, string(dataMap["message"]))
			continue
		}

		for lang, rawValue := range dataMap {
			if err = json.Unmarshal(rawValue, &value); err != nil {
				me.Logger.Error().Stack().Err(err).Msgf("parse (%s) value [%s]", lang, rawValue)
				continue
			}

			var metric matrix.Metric

			if metric = me.Matrix.GetMetric("language." + lang); metric == nil {
				if metric, err = me.Matrix.NewMetricUint64("language." + lang); err != nil {
					return nil, err
				}
				metric.SetName("language")
				metric.SetLabel("lang", lang)
			}

			if err = metric.SetValueUint64(instance, value); err != nil {
				me.Logger.Error().Stack().Err(err).Msgf("set (language.%s) value [%d]", lang, value)
			}
		}

	}

	return me.Matrix, nil

}
