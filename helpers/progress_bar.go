package helpers

import (
	"github.com/cheggaaa/pb"
	"time"
)

func NewPbForBytes(bytes int64) *pb.ProgressBar {
	bar := pb.New64(bytes)
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
	bar.ShowBar = false
	bar.NotPrint = true
	bar.SetRefreshRate(15 * time.Second)
	bar.SetWidth(80)
	bar.SetUnits(pb.U_BYTES)
	bar.Callback = func(out string) {
		println(out)
	}
	return bar
}
