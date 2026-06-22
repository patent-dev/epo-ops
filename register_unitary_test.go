package epo_ops

import "testing"

// TestParseRegisterUnitaryStatuses verifies the unitary-patent (UPP) status
// timeline is parsed off the GetRegisterUNIP record. The fixture is the real EPO
// register-UNIP response for EP3970521, whose request for unitary effect was
// filed (2024-01-05) and then rejected (2024-04-05).
func TestParseRegisterUnitaryStatuses(t *testing.T) {
	docs, err := ParseRegister(string(loadTestData("register-unip-ep3970521.xml")))
	if err != nil {
		t.Fatalf("ParseRegister: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("no register documents parsed")
	}

	got := docs[0].UnitaryStatuses
	if len(got) != 2 {
		t.Fatalf("UnitaryStatuses = %d, want 2: %+v", len(got), got)
	}

	want := []UnitaryStatus{
		{ChangeDate: "20240405", Code: "5", Text: "Request for unitary effect rejected"},
		{ChangeDate: "20240105", Code: "6", Text: "Request for unitary effect filed"},
	}
	for i, w := range want {
		if got[i].ChangeDate != w.ChangeDate || got[i].Code != w.Code || got[i].Text != w.Text {
			t.Errorf("UnitaryStatuses[%d] = %+v, want %+v", i, got[i], w)
		}
	}
}

// TestParseRegisterUnitaryStatusesAbsent confirms a non-unitary register record
// yields no unitary statuses (the field stays empty, never errors).
func TestParseRegisterUnitaryStatusesAbsent(t *testing.T) {
	docs, err := ParseRegister(string(loadTestData("register-biblio.xml")))
	if err != nil {
		t.Fatalf("ParseRegister: %v", err)
	}
	if len(docs) > 0 && len(docs[0].UnitaryStatuses) != 0 {
		t.Errorf("non-unitary record has %d unitary statuses, want 0", len(docs[0].UnitaryStatuses))
	}
}
