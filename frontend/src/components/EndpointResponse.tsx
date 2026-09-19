type EndpointResponseProps = {
  data: unknown;
};

export default function EndpointResponse({ data }: EndpointResponseProps) {
  return (
    <details
      className="group mt-6 rounded-[10px] border border-[#e4e4e7] bg-white"
      open
    >
      <summary className="flex cursor-pointer list-none items-center gap-2 px-6 py-4 text-sm font-[650] [&::-webkit-details-marker]:hidden">
        <span
          className="text-[10px] text-[#4f46e5] transition-transform group-open:rotate-90"
          aria-hidden="true"
        >
          ▶
        </span>
        Endpoint response
        <span className="font-mono text-[11px] font-normal text-[#71717a]">
          POST /detect
        </span>
      </summary>
      <pre className="max-h-90 overflow-auto px-6 pb-5 text-xs leading-[1.6] whitespace-pre-wrap wrap-anywhere">
        {data === null
          ? "Upload accepted. No response body."
          : JSON.stringify(data, null, 2)}
      </pre>
    </details>
  );
}
