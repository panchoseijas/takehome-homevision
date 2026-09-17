package httpapi

import "github.com/panchoseijas/homevision/backend/internal/vision"

// DetectResponse is the JSON body of a successful POST /detect.
type DetectResponse struct {
	Boxes []BoxResponse `json:"boxes"`
}

// BoxResponse is one detected checkbox. BBox is [x1, y1, x2, y2] in pixels of
// the uploaded image, origin top-left, with exclusive right and bottom edges.
// Debug is only populated when the request asked for ?debug=1.
type BoxResponse struct {
	BBox      [4]int         `json:"bbox"`
	IsChecked bool           `json:"is_checked"`
	Debug     *DebugResponse `json:"debug,omitempty"`
}

// DebugResponse mirrors vision.Debug with JSON field names.
type DebugResponse struct {
	FillRatio    float64 `json:"fill_ratio"`
	InkPixels    int     `json:"ink_pixels"`
	InteriorArea int     `json:"interior_area"`
	BorderPx     [4]int  `json:"border_px"`
}

// NewDetectResponse converts detector output to the wire format. An empty
// result encodes as {"boxes":[]} rather than null.
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
