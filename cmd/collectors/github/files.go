package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"goharvest2/pkg/errors"
	"goharvest2/pkg/matrix"
	"goharvest2/pkg/set"
	"strconv"
	"strings"
)

// when counting lines in source-code, we want to ignore these
var ignoreFiles = set.NewFrom([]string{
	"bin",
	"cert",
	".clabot",
	".git",
	".github",
	"go.mod",
	"go.sum",
	"log",
	"pid",
	"vendor",
})

type Content struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Size        int `json:"size"`
	Type        string `json:"type"`
	Content		string
}


func (me *Github) PollFiles() (*matrix.Matrix, error) {

	me.Matrix.Reset()

	// scan Root directory
	err := me.scanDirectory("")

	return me.Matrix, err

}

func (me *Github) scanDirectory(dirPath string) error {

	var (
		rawData		[]byte
		respCode	int
		contentList []Content
		extension string
		size, count	int
		instance *matrix.Instance
		err			error
	)

	if respCode, rawData, err = me.DoRequest("/contents/"+dirPath); err != nil {
		me.Logger.Error().Err(err).Msgf("Requested [%s]", dirPath)
		return err
	}

	if respCode != 200 {
		me.Logger.Warn().Msgf("Requested: [%s] Response: %d", dirPath, respCode)
		return errors.New(errors.API_REQ_REJECTED, strconv.Itoa(respCode))
	}

	if err = json.Unmarshal(rawData, &contentList); err != nil {
		me.Logger.Error().Err(err).Msgf("Unmarshal [%s] data", dirPath)
		return err
	}

	for _, c := range contentList {

		if ignoreFiles.Has(c.Name) {
			continue
		}

		if c.Type == "dir" {
			if err = me.scanDirectory(c.Path); err != nil {
				return err
			}
			continue
		}

		// use file extension as label
		if extension = getExtension(c.Name); extension == "" {
			me.Logger.Debug().Msgf("Skip file [%s], no extension parsed", c.Name)
			continue
		}

		if size, count, err = me.getFileSize(c.Path); err != nil {
			me.Logger.Error().Err(err).Msgf("getLineCount [%s]", c.Path)
			return err
		}

		if instance = me.Matrix.GetInstance(c.Path); instance == nil {
			if instance, err = me.Matrix.NewInstance(c.Path); err != nil {
				me.Logger.Error().Err(err).Msgf("NewInstance [%s]", c.Path)
				return err
			}
		}

		instance.SetLabel("dir", dirPath)
		instance.SetLabel("file", c.Name)
		instance.SetLabel("path", c.Path)
		instance.SetLabel("ext", extension)

		if me.sizeMetric.SetValueInt(instance, size); err != nil {
			return err
		}

		if me.countMetric.SetValueInt(instance, count); err != nil {
			return err
		}

		me.Logger.Debug().Msgf(" + (%s) (%s) [%d bytes] [%d lines]", c.Path, extension, size, count)
	}

	return nil
}

func (me *Github) getFileSize(fp string) (int, int, error) {

	var (
		code int
		data []byte
		err error
		content Content
	)

	if code, data, err = me.DoRequest("/contents/"+fp); err != nil {
		return 0, 0, err
	} 
	
	if code != 200 {
		return 0, 0, errors.New(errors.API_REQ_REJECTED, strconv.Itoa(code))
	} 
	
	if err = json.Unmarshal(data, &content); err != nil {
		return 0, 0, err
	}

	if data, err = base64.StdEncoding.DecodeString(content.Content); err != nil {
		return 0, 0, err
	}

	return content.Size, bytes.Count(data, []byte{'\n'}), nil
}

func getExtension(fileName string) string {
	if split := strings.Split(fileName, "."); len(split) > 1 {
		return split[len(split)-1]
	}
	return ""
}