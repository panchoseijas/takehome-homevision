type EndpointResponseProps = {
  data: unknown;
};

export default function EndpointResponse({ data }: EndpointResponseProps) {
  return (
    <section className="mt-6 rounded-[10px] border border-[#e4e4e7] bg-white p-6">
      <h2 className="text-sm font-[650]">Endpoint response</h2>
      <pre className="mt-4 max-h-90 overflow-auto text-xs leading-[1.6] whitespace-pre-wrap wrap-anywhere">
        {data === null
          ? "Upload accepted. No response body."
          : JSON.stringify(data, null, 2)}
      </pre>
    </section>
  );
}
