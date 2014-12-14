package cuckoof

import (
	"fmt"
	"testing"
)

func TestFilter(t *testing.T) {

	cf := New(8)

	for i := 0; i < 4*6; i++ {
		k := fmt.Sprintf("foo%d", i)
		if ok := cf.Insert([]byte(k)); !ok {
			t.Errorf("Insert(%q) failed", k)
		}
	}
	for i := 0; i < 4*6; i++ {
		k := fmt.Sprintf("foo%d", i)
		if ok := cf.Lookup([]byte(k)); !ok {
			t.Errorf("Lookup(%q) failed", k)
		}
	}

	if ok := cf.Lookup([]byte(`zot`)); ok {
		t.Errorf("Lookup(zot)=true, wrong")
	}

	if ok := cf.Delete([]byte(`zot`)); ok {
		t.Errorf("Delete(zot)=true, wrong")
	}

	if ok := cf.Delete([]byte(`foo0`)); !ok {
		t.Errorf("Delete(foo0)=false, wrong")
	}

	if ok := cf.Lookup([]byte(`foo0`)); ok {
		t.Errorf("Lookup(f00)=true, wrong")
	}
}
