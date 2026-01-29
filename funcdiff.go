package redefine

import (
	"errors"
	"fmt"
	"reflect"
)

type funcDifferences struct {
	In  []*argDifference
	Out []*argDifference
}

func (d *funcDifferences) Error() error {
	errs := []error{}
	for i, arg := range d.In {
		if arg != nil {
			errs = append(errs, fmt.Errorf("argument %d: %v != %v", i, arg.A, arg.B))
		}
	}
	for i, out := range d.Out {
		if out != nil {
			errs = append(errs, fmt.Errorf("output %d: %v != %v", i, out.A, out.B))
		}
	}

	return errors.Join(errs...)
}

type argDifference struct {
	A reflect.Type
	B reflect.Type
}

func diffFuncs(a, b reflect.Value) *funcDifferences {
	at := a.Type()
	bt := b.Type()
	diff := funcDifferences{}

	var inMax int
	if at.NumIn() < bt.NumIn() {
		diff.In = make([]*argDifference, bt.NumIn())
		inMax = at.NumIn() - 1
		for i := at.NumIn(); i < bt.NumIn(); i++ {
			diff.In[i] = &argDifference{
				A: nil,
				B: bt.In(i),
			}
		}
	} else if bt.NumIn() < at.NumIn() {
		diff.In = make([]*argDifference, at.NumIn())
		inMax = bt.NumIn() - 1
		for i := bt.NumIn(); i < at.NumIn(); i++ {
			diff.In[i] = &argDifference{
				A: at.In(i),
				B: nil,
			}
		}
	} else {
		inMax = at.NumIn()
		diff.In = make([]*argDifference, at.NumIn())
	}

	for i := 0; i < inMax; i++ {
		if at.In(i) != bt.In(i) {
			diff.In[i] = &argDifference{
				A: at.In(i),
				B: bt.In(i),
			}
		}
	}

	var outMax int
	if at.NumOut() < bt.NumOut() {
		diff.Out = make([]*argDifference, bt.NumOut())
		outMax = at.NumOut() - 1
		for i := at.NumOut(); i < bt.NumOut(); i++ {
			diff.Out[i] = &argDifference{
				A: nil,
				B: bt.Out(i),
			}
		}
	} else if bt.NumOut() < at.NumOut() {
		diff.Out = make([]*argDifference, at.NumOut())
		outMax = bt.NumOut() - 1
		for i := bt.NumOut(); i < at.NumOut(); i++ {
			diff.Out[i] = &argDifference{
				A: at.Out(i),
				B: nil,
			}
		}
	} else {
		outMax = at.NumOut()
		diff.Out = make([]*argDifference, at.NumOut())
	}

	for i := 0; i < outMax; i++ {
		if at.Out(i) != bt.Out(i) {
			diff.Out[i] = &argDifference{
				A: at.Out(i),
				B: bt.Out(i),
			}
		}
	}

	return &diff
}
