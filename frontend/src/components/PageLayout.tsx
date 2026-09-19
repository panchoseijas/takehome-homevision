import type { ReactNode } from "react";

type PageLayoutProps = {
  children: ReactNode;
};

const COLUMN = "mx-auto w-full max-w-page";

const PAGES = [
  { href: "/", label: "Detection" },
  { href: "/annotate", label: "Annotation editor" },
];

export default function PageLayout({ children }: PageLayoutProps) {
  return (
    <div className="page-backdrop min-h-svh border-t-[5px] border-t-[#4f46e5] px-5 min-[761px]:px-12">
      <header className={`${COLUMN} flex h-20 items-center justify-between`}>
        <a
          className="inline-flex items-center"
          href="/"
          aria-label="HomeVision home"
        >
          <img
            src="/homevision-logo.svg"
            alt="HomeVision"
            width="124"
            height="40"
          />
        </a>
        <nav className="flex gap-2" aria-label="Pages">
          {PAGES.map(({ href, label }) => (
            <a
              key={href}
              className={`rounded-lg border px-2.5 py-2 text-[10px] font-semibold min-[761px]:px-4 min-[761px]:text-xs ${href === window.location.pathname ? "border-[#c7d2fe] bg-[#eef2ff] text-[#4f46e5]" : "border-[#e4e4e7] bg-white text-[#52525b] hover:bg-[#eef2ff]"}`}
              href={href}
              aria-current={
                href === window.location.pathname ? "page" : undefined
              }
            >
              {label}
            </a>
          ))}
        </nav>
      </header>
      <main
        className={`${COLUMN} bg-[linear-gradient(to_right,#e4e4ed66_1px,transparent_1px)] bg-size-[50%_100%] pt-11 pb-9 min-[761px]:bg-size-[25%_100%] min-[761px]:pt-16 min-[761px]:pb-14`}
      >
        {children}
      </main>
      <footer
        className={`${COLUMN} flex flex-wrap justify-between gap-4 border-t border-[#e4e4e7] py-6 text-[11px] text-[#71717a]`}
      >
        <span>
          HomeVision <span className="px-2.5 text-[#d4d4d8]">/</span> Document
          workspace
        </span>
        <span>PNG & JPEG supported</span>
      </footer>
    </div>
  );
}
