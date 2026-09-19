import type { ReactNode } from "react";

type PageLayoutProps = {
  children: ReactNode;
};

const COLUMN = "mx-auto w-full max-w-page";

export default function PageLayout({ children }: PageLayoutProps) {
  return (
    <div className="page-backdrop min-h-svh border-t-[5px] border-t-[#4f46e5] px-5 min-[761px]:px-12">
      <header className={`${COLUMN} flex h-16 items-center justify-between`}>
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
        <p className="text-[10px] tracking-[2px] text-[#71717a] min-[761px]:text-xs">
          CHECKBOX DETECTION
        </p>
      </header>
      <main className={`${COLUMN} pt-4 pb-8 min-[761px]:pt-6`}>{children}</main>
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
