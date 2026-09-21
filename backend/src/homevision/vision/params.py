from dataclasses import dataclass


@dataclass(frozen=True)
class Params:
    max_pixels: int = 25_000_000

    # Odd neighborhood size for adaptive thresholding. It must be larger than
    # the thickest stroke of interest so the local mean is dominated by paper
    # rather than ink.
    adaptive_block_size: int = 31

    # Subtracted from the local mean before comparison. Higher values ignore
    # light shading edges and JPEG noise; lower values keep fainter strokes.
    adaptive_c: float = 15

    # Length of the 1-px-wide horizontal and vertical opening kernels that
    # isolate straight ruling from text and marks.
    line_kernel_length: int = 12

    # Bounds for the outer side length of a candidate.
    min_box_side: int = 20
    max_box_side: int = 120

    # Smallest interior (inside the border) that can hold a legible mark.
    # Specks enclosed by thick rules fall below it.
    min_interior_side: int = 10

    # Bounds width/height (and height/width) of the outer box.
    max_aspect_ratio: float = 1.25

    # Minimum ratio between the interior contour area and its bounding-rectangle area.
    min_rectangularity: float = 0.85

    # Caps the measured border width when expanding an interior hole to the outer box edge.
    max_border_thickness: int = 8

    # Minimum share of the outer box area taken by the interior.
    min_interior_fraction: float = 0.5

    # Fraction of the shorter interior side trimmed from each edge before measuring ink.
    interior_margin: float = 0.12

    # Minimum ink fraction of the trimmed interior for a box to be reported as checked.
    fill_threshold: float = 0.04

    # Intersection-over-union above which two candidates are considered the same box.
    dedupe_iou: float = 0.7
