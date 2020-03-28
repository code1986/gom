package gom

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
)

type varQuery struct {
	Name   string `yaml:"name"`
	SQL    string `yaml:"sql"`
	Fields []string
}

var abbreviationPattern *regexp.Regexp

func init() {
	abbreviationPattern = regexp.MustCompile("<[0-9a-zA-Z_]*>")
}

func (s *varQuery) mapping(key string) string {
	s.Fields = append(s.Fields, key)
	return "?"
}

func (s *varQuery) init(c *config) error {
	s.SQL = os.Expand(s.SQL, s.mapping)

	// find abbreviation and replace sql
	abbrevs := abbreviationPattern.FindAllString(s.SQL, -1)
	if len(abbrevs) > 0 {
		//oldSQL := s.SQL
		for _, abb := range abbrevs {
			abbIn := abb[1 : len(abb)-1]
			if v, ok := c.Abbreviation[abbIn]; ok {
				s.SQL = strings.ReplaceAll(s.SQL, abb, v)
			}
		}
		//fmt.Printf("--------------------------\n%s: abbreviation expand sql from \n>>>>>> %s\n>>> %s\n", s.Name, oldSQL, s.SQL)
	}

	return nil
}

func (s *varQuery) getFieldArgs(v reflect.Value, FieldMap map[string]string) ([]interface{}, error) {
	var ret []interface{}
	var tp reflect.Type

	if v.Kind() == reflect.Ptr {
		tp = v.Elem().Type()
	} else {
		tp = v.Type()
	}

	fieldNum := tp.NumField()

	for _, key := range s.Fields {
		f := v.FieldByName(key)
		if f.IsValid() {
			ret = append(ret, f.Interface())
			continue
		}

		if mapField, ok := FieldMap[key]; ok {
			f := v.FieldByName(mapField)
			if f.IsValid() {
				ret = append(ret, f.Interface())
				continue
			}
		}

		found := false

		// if key is like "tag_id", alterKey will be "tagid"
		var alterKey string
		if strings.Contains(key, "_") {
			alterKey = strings.ReplaceAll(key, "_", "")
		}

		for i := 0; i < fieldNum; i++ {
			f := tp.Field(i)
			if strings.EqualFold(f.Name, key) || strings.EqualFold(f.Name, alterKey) {
				ret = append(ret, v.FieldByName(f.Name).Interface())
				found = true
				break
			}
		}

		if found {
			continue
		}

		return nil, fmt.Errorf("not found key [%s] in struct %s", key, v.Type().Name())
	}
	return ret, nil
}

type sqlList []*varQuery

func (sl sqlList) init(c *config) error {
	for _, s := range sl {
		if err := s.init(c); err != nil {
			return err
		}
	}

	return nil
}

func (sl sqlList) find(name string) *varQuery {
	for _, s := range sl {
		if s.Name == name {
			return s
		}
	}

	return nil
}
