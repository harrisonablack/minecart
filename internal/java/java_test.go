package java

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    Info
		wantErr bool
	}{
		{
			name: "temurin",
			output: `Property settings:
    java.version = 21.0.11
    java.vm.vendor = Eclipse Adoptium
`,
			want: Info{Version: "21.0.11", Vendor: "Eclipse Adoptium"},
		},
		{
			name: "corretto",
			output: `Property settings:
    java.version = 21.0.2
    java.vm.vendor = Amazon.com Inc.
`,
			want: Info{Version: "21.0.2", Vendor: "Amazon.com Inc."},
		},
		{
			name: "zulu",
			output: `Property settings:
    java.version = 21.0.2
    java.vm.vendor = Azul Systems, Inc.
`,
			want: Info{Version: "21.0.2", Vendor: "Azul Systems, Inc."},
		},
		{
			name: "oracle",
			output: `Property settings:
    java.version = 21.0.2
    java.vm.vendor = Oracle Corporation
`,
			want: Info{Version: "21.0.2", Vendor: "Oracle Corporation"},
		},
		{
			name: "echoed java tool options",
			output: `Picked up JAVA_TOOL_OPTIONS: -Dapp.java.version=legacy -Xmx2G
Property settings:
    java.version = 21.0.11
    java.vm.vendor = Eclipse Adoptium
`,
			want: Info{Version: "21.0.11", Vendor: "Eclipse Adoptium"},
		},
		{
			name: "missing version",
			output: `Property settings:
    java.vm.vendor = Eclipse Adoptium
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(tt.output)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parse() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parse() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
