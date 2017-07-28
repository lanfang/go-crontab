package utils

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

//Key的长度必须为8
type IdGenerator struct {
	once       sync.Once
	Key        []byte
	prev       time.Time
	desBlocker cipher.Block
}

var defaultIdGen = IdGenerator{
	Key: []byte("12345678"),
}

func (p *IdGenerator) lazyInit() {
	if p.desBlocker == nil {
		p.prev, _ = time.Parse("20060102150405", "20150701000000")
		p.desBlocker, _ = des.NewCipher(p.Key)
	}
}

func GenCryptoUniqueId() string {
	return defaultIdGen.GenCryptoUniqueId()
}

func (p *IdGenerator) GenCryptoUniqueId() string {
	p.once.Do(p.lazyInit)

	buf := bytes.NewBuffer([]byte{})
	byts1 := make([]byte, 8)
	byts2 := make([]byte, 2)

	rand.Read(byts2)
	var x uint32
	binary.Read(bytes.NewBuffer(byts2), binary.BigEndian, &x)

	pass := uint64((time.Now().UnixNano() - p.prev.UnixNano()) / int64(time.Microsecond))
	binary.Write(buf, binary.BigEndian, pass)
	copy(byts1, buf.Bytes())

	p.desBlocker.Encrypt(byts1, byts1)

	binary.Read(bytes.NewBuffer(byts1), binary.BigEndian, &pass)

	return strings.TrimSpace(fmt.Sprintf("%16d%-16d", x, pass))
}

var typeOfTime = reflect.TypeOf(time.Time{})

//Be careful to use, from,to must be pointer
func DumpStruct(to interface{}, from interface{}) {
	fromv := reflect.ValueOf(from)
	tov := reflect.ValueOf(to)
	if fromv.Kind() != reflect.Ptr || tov.Kind() != reflect.Ptr {
		return
	}

	from_val := reflect.Indirect(fromv)
	to_val := reflect.Indirect(tov)

	for i := 0; i < from_val.Type().NumField(); i++ {
		fdi_from_val := from_val.Field(i)
		fd_name := from_val.Type().Field(i).Name
		fdi_to_val := to_val.FieldByName(fd_name)

		if !fdi_to_val.IsValid() || fdi_to_val.Kind() != fdi_from_val.Kind() {
			continue
		}

		switch fdi_from_val.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if fdi_to_val.Type() != fdi_from_val.Type() {
				fdi_to_val.Set(fdi_from_val.Convert(fdi_to_val.Type()))
			} else {
				fdi_to_val.Set(fdi_from_val)
			}
		case reflect.Slice:
			if fdi_to_val.IsNil() {
				fdi_to_val.Set(reflect.MakeSlice(fdi_to_val.Type(), fdi_from_val.Len(), fdi_from_val.Len()))
			}
			DumpList(fdi_to_val.Interface(), fdi_from_val.Interface())
		case reflect.Struct:
			if fdi_to_val.Type() == typeOfTime {
				if fdi_to_val.Type() != fdi_from_val.Type() {
					continue
				}
				fdi_to_val.Set(fdi_from_val)
			} else {
				DumpStruct(fdi_to_val.Addr(), fdi_from_val.Addr())
			}
		default:
			if fdi_to_val.Type() != fdi_from_val.Type() {
				continue
			}
			fdi_to_val.Set(fdi_from_val)
		}
	}
}

func DumpList(to interface{}, from interface{}) {
	raw_to := reflect.ValueOf(to)
	//raw_from := reflect.ValueOf(from)

	val_from := reflect.Indirect(reflect.ValueOf(from))
	val_to := reflect.Indirect(reflect.ValueOf(to))

	if !(val_from.Kind() == reflect.Slice) || !(val_to.Kind() == reflect.Slice) {
		return
	}

	if raw_to.Kind() == reflect.Ptr && raw_to.Elem().Len() == 0 {
		val_to.Set(reflect.MakeSlice(val_to.Type(), val_from.Len(), val_from.Len()))
	}

	if val_from.Len() == val_to.Len() {
		for i := 0; i < val_from.Len(); i++ {
			switch val_from.Index(i).Kind() {
			case reflect.Struct:
				DumpStruct(val_to.Index(i).Addr().Interface(), val_from.Index(i).Addr().Interface())
			case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.String:
				val_to.Index(i).Set(val_from.Index(i))
			default:
				continue
			}
		}
	}
}
