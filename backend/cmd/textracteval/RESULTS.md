# AWS Textract vs. the local detector

An experiment: run AWS Textract `AnalyzeDocument` with the FORMS feature over
the four testdata images and score its `SELECTION_ELEMENT` blocks against the
hand-made annotations, using the same greedy IoU >= 0.5 matching as
`TestDetectSamples`. The raw responses are cached in `testdata/textract/`
(one call per image, ~$0.05/page, ~$0.20 total), so re-running the eval makes
no AWS calls.

## Scores against the truth annotations (289 boxes)

| Sample | Truth | Textract found | Missed | False pos. | Wrong state |
|---|---|---|---|---|---|
| sample1-urar-page1 | 119 | 98 | 21 | 0 | 0 |
| sample2-neighborhood-site-crop | 43 | 38 | 5 | 1 | 1 |
| sample3-market-conditions-addendum | 48 | 48 | 0 | 0 | 0 |
| sample4-manufactured-home-report | 79 | 71 | 8 | 0 | 0 |
| **Textract total** | **289** | **255 (88.2% recall)** | **34** | **1** | **1** |
| **Local detector** (per `TestDetectSamples`) | **289** | **287 (99.3% recall)** | **2** | **0** | **0** |

## Observations

- Textract's checked/unchecked judgement is excellent on the boxes it finds:
  254 of 255 states correct (99.6%). Detection recall is its weak spot.
- Misses cluster in dense grid regions: sample 1 loses whole runs of boxes in
  the rows around y=2205-2408, and sample 4 similarly drops boxes inside
  tightly packed tables. Textract's FORMS model appears to skip selection
  elements it cannot associate with a key/value pair.
- On sample 2, Textract and the local detector fail on different boxes.
  Textract finds both of the detector's two known misses: the faint
  "Neighborhood Boundaries" box is detected correctly, and the hatched
  "Electricity - Public" box is detected but misread as checked (the one
  wrong state). Its own five misses are ordinary boxes in the utilities /
  off-site improvements rows, including the clearly checked "Electricity -
  Other" box, and its one false positive is the hand-drawn flag-like mark
  next to the Water "Other" box, which the annotations deliberately ignore.
- The local detector beats Textract on every sample; Textract only ties on
  sample 3 (both perfect).

## Reproducing

The responses were fetched once with the AWS CLI (`aws textract
analyze-document --feature-types FORMS --cli-input-json ...`, region
us-east-1, image bytes inline). To score them:

    go run ./cmd/textracteval        # from backend/, testdata path optional arg
