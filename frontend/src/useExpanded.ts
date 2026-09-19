import { useEffect, useState } from "react";

export function useExpanded({
  collapseOnEscape,
}: {
  collapseOnEscape: boolean;
}) {
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    if (!expanded) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previous;
    };
  }, [expanded]);

  useEffect(() => {
    if (!expanded || !collapseOnEscape) return;
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") setExpanded(false);
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [expanded, collapseOnEscape]);

  return { expanded, toggle: () => setExpanded(!expanded) };
}
