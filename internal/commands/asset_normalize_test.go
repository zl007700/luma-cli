package commands

import "testing"

func TestTarget1080Size(t *testing.T) {
	cases := []struct {
		name       string
		width      int
		height     int
		wantWidth  int
		wantHeight int
	}{
		{name: "portrait", width: 720, height: 1280, wantWidth: 1080, wantHeight: 1920},
		{name: "landscape", width: 1280, height: 720, wantWidth: 1920, wantHeight: 1080},
		{name: "square defaults portrait", width: 1080, height: 1080, wantWidth: 1080, wantHeight: 1920},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotWidth, gotHeight := target1080Size(tc.width, tc.height)
			if gotWidth != tc.wantWidth || gotHeight != tc.wantHeight {
				t.Fatalf("target size = %dx%d, want %dx%d", gotWidth, gotHeight, tc.wantWidth, tc.wantHeight)
			}
		})
	}
}

func TestIsVideoAssetFile(t *testing.T) {
	if !isVideoAssetFile("demo.MOV") {
		t.Fatal("expected MOV to be a video asset")
	}
	if isVideoAssetFile("demo.png") {
		t.Fatal("expected PNG not to be a video asset")
	}
}
