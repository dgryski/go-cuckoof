package cuckoof

import (
	"encoding/binary"
	"fmt"
	"math"
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

func TestBasicUint32(t *testing.T) {
	loadFactors := []float64{0.25, 0.5, 0.75, 0.95, 0.97, 0.99, 1.25}
	for p := 4; p <= 16; p += 4 {
		for _, lf := range loadFactors {
			size := 1 << uint16(p) // Total capacity
			r := hammer(size, lf)
			// We tried to insert size * lf elements, size * lf * r.fails failed.
			// Thus, effective load is (size * lf - size * lf * r.fails ) / size.
			// size * lf * r.fails elements were kicked out, so the actual
			// false negatives rate is r.failes. Some of those are masked because of false positives.
			effectiveLoad := lf * (1 - r.fails)
			estimatedFalseNegatives := r.fails * (1 - r.falsePositives)
			what := fmt.Sprintf("size: %d(2^%d) load factor: %.02f%% effective load: %0.3f %#v efn: %f delta: %f", size, p, lf, effectiveLoad, r, estimatedFalseNegatives, estimatedFalseNegatives-r.falseNegatives)
			if lf < 0.96 {
				if r.fails != 0 || r.falseNegatives != 0 {
					t.Errorf("Expected failed==0 && falseNegatives==0 --- %s", what)
				}
			}
			if math.Abs(r.falseNegatives-estimatedFalseNegatives) > 0.02 {
				t.Errorf("Expected delta = |falseNegatives - estimatedFalseNegatives| to be small --- %s", what)
			}
			// TODO: make this test adaptive, taking load into account.
			if r.falsePositives > 0.3 {
				t.Errorf("Expected falseNegatives to be less than 0.3 --- %s", what)

			}
			fmt.Println(what)
		}
	}
	return
}

type rates struct {
	fails, falsePositives, falseNegatives float64
}

func hammer(size int, loadFactor float64) rates {
	f := New(size / 4) // bucket size is 4
	num := int(float64(size) * loadFactor)
	elts := make([][]byte, num)
	bts := make([]byte, num*4)
	var r rates
	// Populate the filter.
	for i := range elts {
		b := bts[i*4 : i*4+4]
		binary.BigEndian.PutUint32(b, uint32(i))
		elts[i] = b
		if !f.Insert(b) {
			r.fails += 1.0 / float64(len(elts))
		}
	}
	// Check for false negatives.
	for _, b := range elts {
		if !f.Lookup(b) {
			r.falseNegatives += 1.0 / float64(len(elts))
		}
	}
	// Check for false positives.
	n := size * 4
	elt := make([]byte, 4)
	for i := 0; i < n; i++ {
		binary.BigEndian.PutUint32(elt, uint32(i+num))
		if f.Lookup(elt) {
			r.falsePositives += 1.0 / float64(n)
		}
	}
	return r
}
