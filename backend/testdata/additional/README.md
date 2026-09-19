# Additional appraisal images

Four pages rendered from the public [REALVALS sample appraisal report](https://realvals.com/wp-content/uploads/2019/04/1004_Appraisal_Report_Sample.pdf), retrieved 2026-09-19. [Publisher page](https://realvals.com/uniform-residential-appraisal-report-example/).

These are new document pages, not altered versions of the challenge images. All are RGB PNGs rendered with PyMuPDF at 300 DPI (2550 x 4200), matching the resolution of the original full-page URAR sample. No marks were added or removed.

| Image | Source PDF page (1-based) | Useful content |
| --- | --- | --- |
| `realvals-urar-subject.png` | 3 | Closest match to sample1: subject, neighborhood, site, utilities, improvements; checked and empty boxes. |
| `realvals-urar-sales-comparison.png` | 4 | Dense comparison grid with sparse checkboxes and reconciliation options. |
| `realvals-urar-pud.png` | 6 | Cost/income sections and empty PUD yes/no checkbox groups. |
| `realvals-urar-inspection.png` | 9 | Certification text and inspection checkbox groups. |

Start with `realvals-urar-subject.png` in the frontend or run from `backend/`:

```sh
go run ./cmd/detect testdata/additional/realvals-urar-subject.png
```

These are clean PDF renders, not camera photos or noisy scans. They have no ground-truth bounding boxes or checked-state annotations. Pages from the same report should stay together if splitting training/evaluation data.

Source attribution is provided for provenance; no open redistribution license was verified.
