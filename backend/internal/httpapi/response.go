package httpapi

import "github.com/panchoseijas/takehome-homevision/backend/internal/vision"

type DetectResponse struct {
	Boxes []BoxResponse `json:"boxes"`
}

type BoxResponse struct {
	BBox      [4]int         `json:"bbox"`
	IsChecked bool           `json:"is_checked"`
	Debug     *DebugResponse `json:"debug,omitempty"`
}

type DebugResponse struct {
	FillRatio    float64 `json:"fill_ratio"`
	InkPixels    int     `json:"ink_pixels"`
	InteriorArea int     `json:"interior_area"`
	BorderPx     [4]int  `json:"border_px"`
}

func NewDetectResponse(boxes []vision.Box, includeDebug bool) DetectResponse {
	response := DetectResponse{Boxes: make([]BoxResponse, 0, len(boxes))}
	for _, box := range boxes {
		item := BoxResponse{
			BBox:      [4]int{box.X1, box.Y1, box.X2, box.Y2},
			IsChecked: box.Checked,
		}
		if includeDebug {
			item.Debug = &DebugResponse{
				FillRatio:    box.Debug.FillRatio,
				InkPixels:    box.Debug.InkPixels,
				InteriorArea: box.Debug.InteriorArea,
				BorderPx:     box.Debug.BorderPx,
			}
		}
		response.Boxes = append(response.Boxes, item)
	}
	return response
}
