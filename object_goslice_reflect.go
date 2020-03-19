package goja

import (
	"reflect"
	"strconv"
)

type objectGoSliceReflect struct {
	objectGoReflect
	lengthProp      valueProperty
	sliceExtensible bool
}

func (o *objectGoSliceReflect) init() {
	o.objectGoReflect.init()
	o.class = classArray
	o.prototype = o.val.runtime.global.ArrayPrototype
	o.sliceExtensible = o.value.CanSet()
	o.lengthProp.writable = o.sliceExtensible
	o.updateLen()
	o.baseObject._put("length", &o.lengthProp)
}

func (o *objectGoSliceReflect) updateLen() {
	o.lengthProp.value = intToValue(int64(o.value.Len()))
}

func (o *objectGoSliceReflect) _hasIdx(idx valueInt) bool {
	if idx := int64(idx); idx >= 0 && idx < int64(o.value.Len()) {
		return true
	}
	return false
}

func (o *objectGoSliceReflect) _hasStr(name string) bool {
	if idx := strToIdx64(name); idx >= 0 && idx < int64(o.value.Len()) {
		return true
	}
	return false
}

func (o *objectGoSliceReflect) _getIdx(idx int64) Value {
	if idx < int64(o.value.Len()) {
		v := o.value.Index(int(idx))
		if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
			return nil
		}
		return o.val.runtime.ToValue(v.Interface())
	}
	return nil
}

func (o *objectGoSliceReflect) getIdx(idx valueInt, receiver Value) Value {
	if idx := int64(idx); idx >= 0 && idx < int64(o.value.Len()) {
		return o._getIdx(idx)
	}
	return o.objectGoReflect.getStr(idx.String(), receiver)
}

func (o *objectGoSliceReflect) getStr(name string, receiver Value) Value {
	return o.getStrWithOwnProp(o.getOwnPropStr(name), name, receiver)
}

func (o *objectGoSliceReflect) getOwnPropStr(name string) Value {
	if idx := strToIdx64(name); idx >= 0 {
		if idx < int64(o.value.Len()) {
			return o._getIdx(idx)
		}
		return nil
	}
	if name == "length" {
		return &o.lengthProp
	}
	return o.objectGoReflect.getOwnPropStr(name)
}

func (o *objectGoSliceReflect) getOwnPropIdx(idx valueInt) Value {
	if idx := int64(idx); idx >= 0 && idx < int64(o.value.Len()) {
		return o._getIdx(idx)
	}
	return nil
}

func (o *objectGoSliceReflect) putIdx(idx int64, v Value, throw bool) bool {
	if idx >= int64(o.value.Len()) {
		if !o.sliceExtensible {
			o.val.runtime.typeErrorResult(throw, "Cannot extend a Go unaddressable reflect slice")
			return false
		}
		o.grow(int(idx + 1))
	}
	val, err := o.val.runtime.toReflectValue(v, o.value.Type().Elem())
	if err != nil {
		o.val.runtime.typeErrorResult(throw, "Go type conversion error: %v", err)
		return false
	}
	o.value.Index(int(idx)).Set(val)
	return true
}

func (o *objectGoSliceReflect) grow(size int) {
	newcap := o.value.Cap()
	if newcap < size {
		// Use the same algorithm as in runtime.growSlice
		doublecap := newcap + newcap
		if size > doublecap {
			newcap = size
		} else {
			if o.value.Len() < 1024 {
				newcap = doublecap
			} else {
				for newcap < size {
					newcap += newcap / 4
				}
			}
		}

		n := reflect.MakeSlice(o.value.Type(), size, newcap)
		reflect.Copy(n, o.value)
		o.value.Set(n)
	} else {
		tail := o.value.Slice(o.value.Len(), size)
		zero := reflect.Zero(o.value.Type().Elem())
		for i := 0; i < tail.Len(); i++ {
			tail.Index(i).Set(zero)
		}
		o.value.SetLen(size)
	}
	o.updateLen()
}

func (o *objectGoSliceReflect) shrink(size int) {
	tail := o.value.Slice(size, o.value.Len())
	zero := reflect.Zero(o.value.Type().Elem())
	for i := 0; i < tail.Len(); i++ {
		tail.Index(i).Set(zero)
	}
	o.value.SetLen(size)
	o.updateLen()
}

func (o *objectGoSliceReflect) putLength(v Value, throw bool) bool {
	newLen := int(toLength(v))
	curLen := o.value.Len()
	if newLen > curLen {
		if !o.sliceExtensible {
			o.val.runtime.typeErrorResult(throw, "Cannot extend Go slice")
			return false
		}
		o.grow(newLen)
	} else if newLen < curLen {
		if !o.sliceExtensible {
			o.val.runtime.typeErrorResult(throw, "Cannot shrink Go slice")
			return false
		}
		o.shrink(newLen)
	}
	return true
}

func (o *objectGoSliceReflect) setOwnIdx(idx valueInt, val Value, throw bool) bool {
	if i := int64(idx); i >= 0 {
		if i >= int64(o.value.Len()) {
			if res, ok := o._setForeignIdx(idx, nil, val, o.val, throw); ok {
				return res
			}
		}
		o.putIdx(i, val, throw)
	} else {
		name := idx.String()
		if res, ok := o._setForeignStr(name, nil, val, o.val, throw); !ok {
			o.val.runtime.typeErrorResult(throw, "Can't set property '%s' on Go slice", name)
			return false
		} else {
			return res
		}
	}
	return true
}

func (o *objectGoSliceReflect) setOwnStr(name string, val Value, throw bool) bool {
	if idx := strToIdx64(name); idx >= 0 {
		if idx >= int64(o.value.Len()) {
			if res, ok := o._setForeignStr(name, nil, val, o.val, throw); ok {
				return res
			}
		}
		o.putIdx(idx, val, throw)
	} else {
		if name == "length" {
			return o.putLength(val, throw)
		}
		if res, ok := o._setForeignStr(name, nil, val, o.val, throw); !ok {
			o.val.runtime.typeErrorResult(throw, "Can't set property '%s' on Go slice", name)
			return false
		} else {
			return res
		}
	}
	return true
}

func (o *objectGoSliceReflect) setForeignIdx(idx valueInt, val, receiver Value, throw bool) (bool, bool) {
	return o._setForeignIdx(idx, trueValIfPresent(o._hasIdx(idx)), val, receiver, throw)
}

func (o *objectGoSliceReflect) setForeignStr(name string, val, receiver Value, throw bool) (bool, bool) {
	return o._setForeignStr(name, trueValIfPresent(o._hasStr(name)), val, receiver, throw)
}

func (o *objectGoSliceReflect) hasOwnPropertyIdx(idx valueInt) bool {
	return o._hasIdx(idx)
}

func (o *objectGoSliceReflect) hasOwnPropertyStr(name string) bool {
	if o._hasStr(name) {
		return true
	}
	return o.objectGoReflect._has(name)
}

func (o *objectGoSliceReflect) defineOwnPropertyIdx(idx valueInt, descr PropertyDescriptor, throw bool) bool {
	if idx >= 0 {
		if !o.val.runtime.checkHostObjectPropertyDescr(idx.String(), descr, throw) {
			return false
		}
		val := descr.Value
		if val == nil {
			val = _undefined
		}
		o.putIdx(int64(idx), val, throw)
		return true
	}
	o.val.runtime.typeErrorResult(throw, "Cannot define property '%d' on a Go slice", idx)
	return false
}

func (o *objectGoSliceReflect) defineOwnPropertyStr(name string, descr PropertyDescriptor, throw bool) bool {
	if idx := strToIdx64(name); idx >= 0 {
		if !o.val.runtime.checkHostObjectPropertyDescr(name, descr, throw) {
			return false
		}
		val := descr.Value
		if val == nil {
			val = _undefined
		}
		o.putIdx(idx, val, throw)
		return true
	}
	o.val.runtime.typeErrorResult(throw, "Cannot define property '%s' on a Go slice", name)
	return false
}

func (o *objectGoSliceReflect) toPrimitiveNumber() Value {
	return o.toPrimitiveString()
}

func (o *objectGoSliceReflect) toPrimitiveString() Value {
	return o.val.runtime.arrayproto_join(FunctionCall{
		This: o.val,
	})
}

func (o *objectGoSliceReflect) toPrimitive() Value {
	return o.toPrimitiveString()
}

func (o *objectGoSliceReflect) deleteStr(name string, throw bool) bool {
	if idx := strToIdx64(name); idx >= 0 {
		if idx < int64(o.value.Len()) {
			o.value.Index(int(idx)).Set(reflect.Zero(o.value.Type().Elem()))
		}
		return true
	}

	return o.objectGoReflect.deleteStr(name, throw)
}

func (o *objectGoSliceReflect) deleteIdx(i valueInt, throw bool) bool {
	idx := int64(i)
	if idx >= 0 {
		if idx < int64(o.value.Len()) {
			o.value.Index(int(idx)).Set(reflect.Zero(o.value.Type().Elem()))
		}
	}
	return true
}

type gosliceReflectPropIter struct {
	o          *objectGoSliceReflect
	idx, limit int
}

func (i *gosliceReflectPropIter) next() (propIterItem, iterNextFunc) {
	if i.idx < i.limit && i.idx < i.o.value.Len() {
		name := strconv.Itoa(i.idx)
		i.idx++
		return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.next
	}

	return i.o.objectGoReflect.enumerateUnfiltered()()
}

func (o *objectGoSliceReflect) ownKeys(all bool, accum []Value) []Value {
	for i := 0; i < o.value.Len(); i++ {
		accum = append(accum, asciiString(strconv.Itoa(i)))
	}

	return o.objectGoReflect.ownKeys(all, accum)
}

func (o *objectGoSliceReflect) enumerateUnfiltered() iterNextFunc {
	return (&gosliceReflectPropIter{
		o:     o,
		limit: o.value.Len(),
	}).next
}

func (o *objectGoSliceReflect) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoSliceReflect); ok {
		return o.value.Interface() == other.value.Interface()
	}
	return false
}

func (o *objectGoSliceReflect) sortLen() int64 {
	return int64(o.value.Len())
}

func (o *objectGoSliceReflect) sortGet(i int64) Value {
	return o.getIdx(valueInt(i), nil)
}

func (o *objectGoSliceReflect) swap(i, j int64) {
	ii := valueInt(i)
	jj := valueInt(j)
	x := o.getIdx(ii, nil)
	y := o.getIdx(jj, nil)

	o.setOwnIdx(ii, y, false)
	o.setOwnIdx(jj, x, false)
}
