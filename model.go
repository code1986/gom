package gom

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

type modelImpl struct {
	file   string
	vtype  reflect.Type
	config config
}

var _ Model = (*modelImpl)(nil)

func (m *modelImpl) extractStruct(q *varQuery, arg interface{}) (useBuildArgs bool, buildArgs []any, err error) {
	tp := reflect.TypeOf(arg)
	v := reflect.ValueOf(arg)

	useBuildArgs = false
	if tp == m.vtype {
		useBuildArgs = true
		buildArgs, err = q.getFieldArgs(v, m.config.FieldMap)
	} else if tp.Kind() == reflect.Ptr && tp.Elem() == m.vtype {
		useBuildArgs = true
		buildArgs, err = q.getFieldArgs(v.Elem(), m.config.FieldMap)
	}
	return useBuildArgs, buildArgs, err
}

func (m *modelImpl) QueryRow(c SQLConn, name string, args ...any) (any, error) {
	q := m.config.findQuery(name)
	if q == nil {
		return nil, fmt.Errorf("can't find query [%s] in file [%s]", name, m.file)
	}

	var useFieldArgs bool = false
	var fieldArgs []any
	var result *sql.Row
	var err error

	if len(args) == 1 {
		if useFieldArgs, fieldArgs, err = m.extractStruct(q, args[0]); err != nil {
			return nil, err
		}
	}

	if useFieldArgs {
		result = c.QueryRow(q.SQL, fieldArgs...)
	} else {
		result = c.QueryRow(q.SQL, args...)
	}

	if result == nil {
		return nil, nil
	}

	return DefaultOrm.ToObjByType(result, m.vtype)
}

func (m *modelImpl) Query(c SQLConn, name string, args ...any) ([]any, error) {
	q := m.config.findQuery(name)
	if q == nil {
		return nil, fmt.Errorf("can't find query [%s] in file [%s]", name, m.file)
	}

	var useBuildArgs bool = false
	var fieldArgs []any
	var rows *sql.Rows
	var err error

	if len(args) == 1 {
		if useBuildArgs, fieldArgs, err = m.extractStruct(q, args[0]); err != nil {
			return nil, err
		}
	}

	if useBuildArgs {
		rows, err = c.Query(q.SQL, fieldArgs...)
	} else {
		rows, err = c.Query(q.SQL, args...)
	}

	if err != nil {
		return nil, err
	}

	return DefaultOrm.ToMultiObjsByType(rows, m.vtype)
}

var (
	errArgumentIsNotSlice         = errors.New("argument is not a slice or array")
	errArgumentInSliceIsNotStruct = errors.New("argument in slice is not a struct or ptr to struct")
	insertStartPattern            = regexp.MustCompile(`(?i)^\s*(insert)\s*`)
	valuesPattern                 = regexp.MustCompile(`(?is)(?:VALUES)\s*(\s*[(].*[)]\s*)\s*$`)
)

func buildBatchQuery(insertSQL string, batchSize int) (string, error) {
	if batchSize == 1 {
		return insertSQL, nil
	}

	matched := valuesPattern.FindStringSubmatch(insertSQL)
	if matched == nil {
		return "", fmt.Errorf("not found values part in sql")
	}

	var sb strings.Builder
	sb.WriteString(insertSQL)
	for batchSize > 1 {
		sb.WriteString(",")
		sb.WriteString(matched[1])
		batchSize--
	}

	return sb.String(), nil
}

func (m *modelImpl) MultiInsert(c SQLConn, name string, slice any, batchSize int) (int64, int64, error) {
	tp := reflect.TypeOf(slice)
	if tp.Kind() != reflect.Slice && tp.Kind() != reflect.Array {
		return 0, 0, errArgumentIsNotSlice
	}

	elemTp := tp.Elem()
	if elemTp.Kind() != reflect.Struct &&
		elemTp.Kind() != reflect.Ptr {
		return 0, 0, errArgumentInSliceIsNotStruct
	}

	q := m.config.findExec(name)
	if q == nil {
		return 0, 0, fmt.Errorf("can't find exec [%s] in file [%s]", name, m.file)
	}

	if !insertStartPattern.MatchString(q.SQL) {
		return 0, 0, fmt.Errorf("[%s:%s]: sql is not insert: %s ", m.file, name, q.SQL)
	}

	var batchSQLSize, lastBatchSQLSize, offset int
	var totalAffectRows, lastInsertID int64
	var batchSQL string
	var err error
	values := reflect.ValueOf(slice)
	left := values.Len()
	for left > 0 {
		if left >= batchSize {
			batchSQLSize = batchSize
		} else {
			batchSQLSize = left
		}

		if batchSQLSize != lastBatchSQLSize {
			batchSQL, err = buildBatchQuery(q.SQL, batchSQLSize)
			if err != nil {
				return totalAffectRows, lastInsertID, err
			}
		}

		var args []interface{}
		i := 0
		for i < batchSQLSize {
			v := values.Index(offset)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			fieldArgs, err := q.getFieldArgs(v, m.config.FieldMap)
			if err != nil {
				return totalAffectRows, lastInsertID,
					fmt.Errorf("build argument is the [%d]-nth element failed: %v", offset, err)
			}
			args = append(args, fieldArgs...)
			i++
			offset++
		}

		result, err := c.Exec(batchSQL, args...)
		if err != nil {
			return totalAffectRows, lastInsertID, err
		}

		lastBatchSQLSize = batchSQLSize
		left -= batchSQLSize

		rowCount, _ := result.RowsAffected()
		lastInsertID, _ = result.LastInsertId()
		totalAffectRows += rowCount

		//fmt.Println("batch insert ", rowCount, "rows, sql = ", batchSQL)
	}

	return totalAffectRows, lastInsertID, nil
}

func (m *modelImpl) Exec(c SQLConn, name string, args ...any) (int64, int64, error) {
	q := m.config.findExec(name)
	if q == nil {
		return 0, 0, fmt.Errorf("can't find exec [%s] in file [%s]", name, m.file)
	}

	var useFieldArgs bool = false
	var fieldArgs []any
	var result sql.Result
	var err error

	if len(args) == 1 {
		if useFieldArgs, fieldArgs, err = m.extractStruct(q, args[0]); err != nil {
			return 0, 0, err
		}
	}

	if useFieldArgs {
		result, err = c.Exec(q.SQL, fieldArgs...)
	} else {
		result, err = c.Exec(q.SQL, args...)
	}

	if err != nil {
		return 0, 0, err
	}

	rowCount, _ := result.RowsAffected()
	lastID, _ := result.LastInsertId()
	return rowCount, lastID, nil
}

func findFileInDirs(file string, dirs []string) string {

	if _, err := os.Stat(file); err == nil {
		return file
	}

	for _, dir := range dirs {
		filePath := path.Join(dir, file)
		if _, err := os.Stat(filePath); err == nil {
			return filePath
		}
	}
	return ""
}

var yamlPath []string

// AddYamlPaths add path for search yaml
func AddYamlPaths(paths ...string) {
	yamlPath = append(yamlPath, paths...)
}

// LoadModel ...
func LoadModel(file string, v any) (Model, error) {
	pickFile := findFileInDirs(file, yamlPath)

	if pickFile == "" {
		return nil, fmt.Errorf("not found file %s in pat list: %s", file, yamlPath)
	}

	m := &modelImpl{file: pickFile}

	tp := reflect.TypeOf(v)
	if tp.Kind() == reflect.Ptr && tp.Elem().Kind() == reflect.Struct {
		m.vtype = tp.Elem()
	} else {
		if tp.Kind() == reflect.Struct {
			return nil, fmt.Errorf("second argument is not ptr to a struct, should use '&%s{}'", tp.Name())
		} else {
			return nil, fmt.Errorf("second argument is not ptr to a struct")
		}
	}

	if !reflect.ValueOf(v).MethodByName("Scan").IsValid() {
		return nil, fmt.Errorf("%s not has [Scan] method", m.vtype.Name())
	}

	var fileContent, err = ioutil.ReadFile(pickFile)
	if err != nil {
		return nil, fmt.Errorf("read file [%v] error: %v", file, err)
	}

	if err = yaml.Unmarshal(fileContent, &m.config); err != nil {
		return nil, fmt.Errorf("yaml unmarshal file [%v] error: %v", file, err)
	}

	if err = m.config.init(); err != nil {
		return nil, err
	}

	return m, nil
}
