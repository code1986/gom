package gom

import (
	"database/sql"
	"fmt"
	"reflect"
)

type ormImp struct {
}

func NewOrm() Orm {
	return &ormImp{}
}

var DefaultOrm = NewOrm()

func (o *ormImp) parserType(v interface{}) (reflect.Type, error) {
	tp := reflect.TypeOf(v)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}

	if tp.Kind() != reflect.Struct {
		return nil, fmt.Errorf("interface is not struct or ptr to a struct")
	}

	if !reflect.ValueOf(v).MethodByName("Scan").IsValid() {
		ptr := reflect.New(tp)
		if !ptr.MethodByName("Scan").IsValid() {
			return nil, fmt.Errorf("%s not has [Scan] method", tp.Name())
		}
		fmt.Printf("found *%s.Scan method\n", tp.Name())
	}

	return tp, nil
}

func (o *ormImp) ToObj(row *sql.Row, temp interface{}) (any, error) {
	tp, err := o.parserType(temp)
	if err != nil {
		return nil, err
	}

	return o.ToObjByType(row, tp)
}

func (o *ormImp) ToMultiObjs(rows *sql.Rows, temp interface{}) ([]any, error) {
	tp, err := o.parserType(temp)
	if err != nil {
		return nil, err
	}

	return o.ToMultiObjsByType(rows, tp)
}

func (o *ormImp) scanObject(row reflect.Value, vtype reflect.Type) (any, error) {
	v := reflect.New(vtype)
	face := v.Interface()
	method := v.MethodByName("Scan")
	results := method.Call([]reflect.Value{row})
	if len(results) > 0 && results[0].Interface() != nil {
		return nil, results[0].Interface().(error)
	}
	return face, nil
}

func (o *ormImp) ToObjByType(row *sql.Row, vtype reflect.Type) (any, error) {
	return o.scanObject(reflect.ValueOf(row), vtype)
}

func (o *ormImp) ToMultiObjsByType(rows *sql.Rows, vtype reflect.Type) (result []any, err error) {
	defer rows.Close()
	for rows.Next() {
		obj, err := o.scanObject(reflect.ValueOf(rows), vtype)
		if err != nil {
			return nil, err
		}
		result = append(result, obj)
	}

	return result, nil
}
