package project

import (
	"strings"
	"testing"
)

func TestRequiresStructuredFactBody(t *testing.T) {
	cases := []struct {
		cat, key string
		want     bool
	}{
		{"problem", "note/misc", true},
		{"note", "model/topsis-plan", true},
		{"data", "data/attachment-audit", true},
		{"paper", "x", true},
		{"note", "note/misc", false},
		{"", "result/problem-1", true},
	}
	for _, tc := range cases {
		if got := RequiresStructuredFactBody(tc.cat, tc.key); got != tc.want {
			t.Errorf("RequiresStructuredFactBody(%q,%q)=%v want %v", tc.cat, tc.key, got, tc.want)
		}
	}
}

func TestIsSparseFactBody(t *testing.T) {
	long := strings.Repeat("x", 150)
	if !IsSparseFactBody("model", "model/x", "") {
		t.Error("empty body should be sparse")
	}
	if !IsSparseFactBody("model", "model/x", long) {
		t.Error("body without modeling card clues should be sparse")
	}
	body := "## 模型规格\n- 变量: x\n## 验证方案\n- 基线: 线性模型\n"
	if IsSparseFactBody("model", "model/x", body) {
		t.Error("structured body should not be sparse")
	}
	if IsSparseFactBody("note", "note/x", "") {
		t.Error("note fact empty body is ok")
	}
}
