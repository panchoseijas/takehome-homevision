export type DetectedBox = {
  bbox: [number, number, number, number];
  is_checked: boolean;
};

export type DetectResponse = {
  boxes: DetectedBox[];
};
