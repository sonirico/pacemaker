package pacemaker

import (
	"testing"
	"time"
)

func TestAtLeast(t *testing.T) {
	atLeast1 := AtLeast(1)

	if 1 != atLeast1(0) {
		t.Errorf("unexpected value, want 1, have 0")
	}
}

func TestTimeFromNsStr(t *testing.T) {
	type (
		args struct {
			raw string
		}

		want struct {
			ts  time.Time
			err error
		}

		testCase struct {
			name string
			args args
			want want
		}
	)

	tests := []testCase{
		{
			name: "string ok should pass",
			args: args{
				raw: "1669067236944898357",
			},
			want: want{
				err: nil,
				ts:  newTime("2022-11-21T21:47:16.944898357Z"),
			},
		},
	}

	for _, test := range tests {
		actual, err := TimeFromNsStr(test.args.raw)

		if err != nil && test.want.err == nil {
			t.Errorf("unexpected error, want none have %v", err)
		}
		if err == nil && test.want.err != nil {
			t.Errorf("unexpected error, want %v have none", err)
		}

		if !actual.Equal(test.want.ts) {
			t.Errorf("unexpected time, want %v have %v",
				test.want.ts, actual)
		}
	}
}

func TestLastTsFromKeys(t *testing.T) {
	type (
		args struct {
			keys []string
			sep  string
		}

		want struct {
			ts  time.Time
			err error
		}

		testCase struct {
			name string
			args args
			want want
		}
	)

	tests := []testCase{
		{
			name: "be latest key the last",
			args: args{
				keys: []string{
					"mitest|1669067236944898356",
					"mitest|1669067236944898355",
					"mitest|1669067236944898357",
				},
				sep: "|",
			},
			want: want{
				err: nil,
				ts:  newTime("2022-11-21T21:47:16.944898357Z"),
			},
		},
		{
			name: "be latest key the first",
			args: args{
				keys: []string{
					"mitest|1669067236944898357",
					"mitest|1669067236944898356",
					"mitest|1669067236944898355",
				},
				sep: "|",
			},
			want: want{
				err: nil,
				ts:  newTime("2022-11-21T21:47:16.944898357Z"),
			},
		},
	}

	for _, test := range tests {
		actual, err := LatestTsFromKeys(test.args.keys, test.args.sep)

		if err != nil && test.want.err == nil {
			t.Errorf("unexpected error, want none have %v", err)
		}
		if err == nil && test.want.err != nil {
			t.Errorf("unexpected error, want %v have none", err)
		}

		if !actual.Equal(test.want.ts) {
			t.Errorf("unexpected time, want %v have %v",
				test.want.ts, actual)
		}
	}
}

func newTime(raw string) time.Time {
	ts, err := time.Parse("2006-01-02T15:04:05.000000000Z", raw)
	if err != nil {
		panic(err)
	}
	return ts
}
