package gormmiddleware

import "testing"

type testRow struct {
	Name string
}

type testRowPtr struct {
	Name *string
}

func TestFilterDest_Struct(t *testing.T) {
	rules := []Rule{{Table: "x", Field: "Name", DenyPrefixes: []string{"Device.FaultMgmt."}}}

	allowed := testRow{Name: "Device.DeviceInfo.SerialNumber"}
	if _, replaced := filterDest(allowed, rules); replaced {
		t.Fatalf("expected allowed struct not replaced")
	}

	denied := testRow{Name: "Device.FaultMgmt.Alarm.1.Status"}
	outAny, replaced := filterDest(denied, rules)
	if !replaced {
		t.Fatalf("expected denied struct replaced")
	}
	out, ok := outAny.([]testRow)
	if !ok {
		t.Fatalf("expected []testRow, got %T", outAny)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty slice, got %+v", out)
	}
}

func TestFilterDest_PtrStruct(t *testing.T) {
	rules := []Rule{{Table: "x", Field: "Name", DenyPrefixes: []string{"Device.FaultMgmt."}}}

	allowed := &testRow{Name: "Device.DeviceInfo.SerialNumber"}
	if _, replaced := filterDest(allowed, rules); replaced {
		t.Fatalf("expected allowed ptr struct not replaced")
	}

	denied := &testRow{Name: "Device.FaultMgmt.Alarm.1.Status"}
	outAny, replaced := filterDest(denied, rules)
	if !replaced {
		t.Fatalf("expected denied ptr struct replaced")
	}
	out, ok := outAny.([]testRow)
	if !ok {
		t.Fatalf("expected []testRow, got %T", outAny)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty slice, got %+v", out)
	}
}

func TestFilterDest_Slice(t *testing.T) {
	rules := []Rule{{Table: "x", Field: "Name", DenyPrefixes: []string{"Device.FaultMgmt."}}}

	in := []testRow{
		{Name: "Device.FaultMgmt.A"},
		{Name: "Device.DeviceInfo.A"},
		{Name: "Device.FaultMgmt.B"},
	}

	outAny, replaced := filterDest(in, rules)
	if !replaced {
		t.Fatalf("expected slice replaced")
	}
	out, ok := outAny.([]testRow)
	if !ok {
		t.Fatalf("expected []testRow, got %T", outAny)
	}
	if len(out) != 1 || out[0].Name != "Device.DeviceInfo.A" {
		t.Fatalf("unexpected filtered slice: %+v", out)
	}
}

func TestFilterDest_PtrSlice(t *testing.T) {
	rules := []Rule{{Table: "x", Field: "Name", DenyPrefixes: []string{"Device.FaultMgmt."}}}

	in := &[]testRow{
		{Name: "Device.FaultMgmt.A"},
		{Name: "Device.DeviceInfo.A"},
	}

	outAny, replaced := filterDest(in, rules)
	if replaced {
		t.Fatalf("expected ptr slice not replaced")
	}
	if outAny != in {
		t.Fatalf("expected dest pointer preserved")
	}
	if len(*in) != 1 || (*in)[0].Name != "Device.DeviceInfo.A" {
		t.Fatalf("unexpected filtered ptr slice: %+v", *in)
	}
}

func TestFilterDest_EmptyAfterFilterSkips(t *testing.T) {
	rules := []Rule{{Table: "x", Field: "Name", DenyPrefixes: []string{"Device.FaultMgmt."}}}

	in := []testRow{
		{Name: "Device.FaultMgmt.A"},
		{Name: "Device.FaultMgmt.B"},
	}
	outAny, replaced := filterDest(in, rules)
	if !replaced {
		t.Fatalf("expected empty slice replaced")
	}
	out, ok := outAny.([]testRow)
	if !ok {
		t.Fatalf("expected []testRow, got %T", outAny)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty slice, got %+v", out)
	}

	inPtr := &[]testRow{
		{Name: "Device.FaultMgmt.A"},
	}
	_, replaced = filterDest(inPtr, rules)
	if replaced {
		t.Fatalf("expected empty ptr slice not replaced")
	}
	if len(*inPtr) != 0 {
		t.Fatalf("expected ptr slice emptied")
	}
}

func TestReadStringField_PtrString(t *testing.T) {
	rules := []Rule{{Table: "x", Field: "Name", DenyPrefixes: []string{"Device.FaultMgmt."}}}

	deniedStr := "Device.FaultMgmt.A"
	allowedStr := "Device.DeviceInfo.A"

	denied := testRowPtr{Name: &deniedStr}
	outAny, replaced := filterDest(denied, rules)
	if !replaced {
		t.Fatalf("expected denied ptr string replaced")
	}
	out, ok := outAny.([]testRowPtr)
	if !ok {
		t.Fatalf("expected []testRowPtr, got %T", outAny)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty slice, got %+v", out)
	}

	allowed := testRowPtr{Name: &allowedStr}
	if _, replaced := filterDest(allowed, rules); replaced {
		t.Fatalf("expected allowed ptr string not replaced")
	}
}
