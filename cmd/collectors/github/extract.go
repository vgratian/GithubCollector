package github

import (
    "encoding/json"
	"goharvest2/pkg/errors"
	"goharvest2/pkg/matrix"
)


// extract counter values (metric, labels) from partially parsed json
// and populates matrix
func (me *Github) extractJsonData(rootKey string, rootElem *Element, data map[string]json.RawMessage, key string, labels map[string]string) error {

	var (
		elem					*Element
		instance                *matrix.Instance
        instanceKey             string
		instanceLabels			map[string]string
		err                     error
        subJsons                 []map[string]json.RawMessage
		subJson					map[string]json.RawMessage
	)

	// update instance key and labels
	instanceKey, instanceLabels = me.extractKeyAndLabels(rootElem, data, key, labels)

	// for nested element: we will recursively parse in the child
	for _, elem = range rootElem.Elements {
		if elem.IsNested {
			if err = json.Unmarshal(data[elem.Name], &subJsons); err != nil {
                return err
            }
			for _, subJson = range subJsons {
				if err = me.extractJsonData(rootKey, elem, subJson, instanceKey, instanceLabels); err != nil {
					return err
				}
			}
			return nil
		}
	}

	// if we got here, this is the last element that we need to parse

	// no instance key, means the only instance is the repo itself
	// instance should already be created
	if instanceKey == "" {
		if instance = me.matrices[rootKey].GetInstance("repo"); instance == nil {
			return errors.New("MISSING INSTANCE", "repo")
		}
	} else if instance = me.matrices[rootKey].GetInstance(instanceKey); instance == nil {
		me.Logger.Debug().Msgf("adding new instance [%s] (%v)", instanceKey, instanceLabels)
		if instance, err = me.matrices[rootKey].NewInstance(instanceKey); err != nil {
			me.Logger.Error().Err(err).Msg("new instance")
			return err
		}
	}

	if instanceLabels != nil {
		for label, val := range instanceLabels {
			instance.SetLabel(label, val)
		}
	}

	return me.extractMetricValues(rootKey, rootElem, instance, data)

}

func (me *Github) extractKeyAndLabels(elem *Element, data map[string]json.RawMessage, key string, labels map[string]string) (string, map[string]string) {

	updatedLabels := make(map[string]string)
	updatedKey := key

	if labels != nil {
		for k, v := range labels {
			updatedLabels[k] = v
		}
	}

	for _, e := range elem.Elements {
		value := ""
        if e.IsLabel && json.Unmarshal(data[e.Name], &value) == nil {
            updatedLabels[e.DisplayName] = value
			if e.IsKey {
				updatedKey += "." + value
			}
        }
    }

	return updatedKey, updatedLabels
}

func (me *Github) extractMetricValues(key string, elem *Element, instance *matrix.Instance, data map[string]json.RawMessage) error {

    var (
        m matrix.Metric
        val int64
        err error
    )

	for _, e := range elem.Elements {

        if e.IsMetric {

            if m = me.matrices[key].GetMetric(e.Name); m == nil {
                me.Logger.Warn().Msgf("<%s> missing metric (%s)", key, e.Name)
                return errors.New("MISSING METRIC", e.Name)
            }

            if err = json.Unmarshal(data[e.Name], &val); err != nil {
                me.Logger.Error().Err(err).Msgf("<%s> can't extract value (%s)", key, e.Name)
				return err
            } 
			
			if err = m.SetValueInt64(instance, val); err != nil {
                me.Logger.Error().Err(err).Msgf("<%s> SetValueInt64 (%s) => [%d]", key, e.Name, val)
				return err
            }
        	
			me.Logger.Debug().Msgf("<%s> SetValueInt64 (%s) => [%d]", key, e.Name, val)
        }
    }

	return nil
}
