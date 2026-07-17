package msgid

import "testing"

// The expected values are taken from the TianLong3 client generated enums,
// pinning transformgen to the same deterministic algorithm.
func TestComputeMatchesClientValues(t *testing.T) {
	cases := []struct {
		name     string
		toServer bool
		want     uint32
	}{
		{"MsgCtrResHeartbeat", false, 119979348},
		{"MsgCtrResCreateRole", false, 117401164},
		{"MsgMapReqUseSkill", true, 217082693},
	}
	for _, tc := range cases {
		if got := Compute(tc.name, tc.toServer); got != tc.want {
			t.Fatalf("Compute(%q, %v) = %d, want %d", tc.name, tc.toServer, got, tc.want)
		}
	}
}

func TestComputeIsDeterministic(t *testing.T) {
	if Compute("SomeRequest", true) != Compute("SomeRequest", true) {
		t.Fatal("Compute is not deterministic")
	}
}

func TestComputeBandsBySide(t *testing.T) {
	server := Compute("SampleMessage", true)
	client := Compute("SampleMessage", false)
	if server < 200000000 || server >= 300000000 {
		t.Fatalf("server band id = %d, want [200000000,300000000)", server)
	}
	if client < 100000000 || client >= 200000000 {
		t.Fatalf("client band id = %d, want [100000000,200000000)", client)
	}
}
