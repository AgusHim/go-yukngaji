package migrate

import (
	"strings"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want []string
	}{
		{
			name: "dua pernyataan sederhana",
			sql:  "CREATE TABLE a (id int);\nCREATE TABLE b (id int);",
			want: []string{"CREATE TABLE a (id int)", "CREATE TABLE b (id int)"},
		},
		{
			name: "titik koma di dalam string tidak memisah",
			sql:  "INSERT INTO a VALUES ('satu; dua');\nSELECT 1;",
			want: []string{"INSERT INTO a VALUES ('satu; dua')", "SELECT 1"},
		},
		{
			name: "quote ganda di dalam string",
			sql:  "INSERT INTO a VALUES ('it''s; fine');",
			want: []string{"INSERT INTO a VALUES ('it''s; fine')"},
		},
		{
			name: "komentar baris dibuang",
			sql:  "-- komentar; dengan titik koma\nSELECT 1;",
			want: []string{"SELECT 1"},
		},
		{
			name: "komentar blok dibuang",
			sql:  "SELECT /* ini; komentar */ 1;",
			want: []string{"SELECT   1"},
		},
		{
			name: "titik koma di dalam dollar quote tidak memisah",
			sql:  "DO $$ BEGIN PERFORM 1; PERFORM 2; END $$;",
			want: []string{"DO $$ BEGIN PERFORM 1; PERFORM 2; END $$"},
		},
		{
			name: "dollar quote bertag",
			sql:  "CREATE FUNCTION f() RETURNS int AS $body$ SELECT 1; $body$ LANGUAGE sql;",
			want: []string{"CREATE FUNCTION f() RETURNS int AS $body$ SELECT 1; $body$ LANGUAGE sql"},
		},
		{
			name: "placeholder $1 bukan dollar quote",
			sql:  "SELECT * FROM a WHERE id = $1;",
			want: []string{"SELECT * FROM a WHERE id = $1"},
		},
		{
			name: "identifier berkutip dipertahankan",
			sql:  `SELECT "kolom;aneh" FROM "tabel";`,
			want: []string{`SELECT "kolom;aneh" FROM "tabel"`},
		},
		{
			name: "pernyataan kosong diabaikan",
			sql:  ";;\n  \n;",
			want: nil,
		},
		{
			name: "tanpa titik koma penutup tetap terbaca",
			sql:  "SELECT 1",
			want: []string{"SELECT 1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitStatements(tt.sql)
			if len(got) != len(tt.want) {
				t.Fatalf("jumlah pernyataan = %d (%q), want %d (%q)", len(got), got, len(tt.want), tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("pernyataan[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitStatementsKeepsCommentFreeStatementIntact(t *testing.T) {
	// Komentar di antara dua baris tidak boleh menyatukan dua pernyataan.
	sql := "SELECT 1; -- komentar\nSELECT 2;"
	got := SplitStatements(sql)
	if len(got) != 2 {
		t.Fatalf("harus 2 pernyataan, dapat %d: %q", len(got), got)
	}
	if strings.TrimSpace(got[0]) != "SELECT 1" || strings.TrimSpace(got[1]) != "SELECT 2" {
		t.Errorf("hasil tak terduga: %q", got)
	}
}
