import type { ReactNode } from "react";

type PageLayoutProps = {
  children: ReactNode;
};

export default function PageLayout({ children }: PageLayoutProps) {
  return (
    <div className="page-shell">
      <header className="site-header">
        <a
          className="inline-flex items-center"
          href="/"
          aria-label="HomeVision home"
        >
          <img src="/homevision-logo.svg" alt="HomeVision" width="124" height="40" />
        </a>
        <span className="workspace-label">Document workspace</span>
      </header>
      <main className="workspace-main">{children}</main>
      <footer className="site-footer">
        <span>
          HomeVision <span className="px-2.5 text-[#d4d4d8]">/</span> Document
          workspace
        </span>
        <span>PNG & JPEG supported</span>
      </footer>
    </div>
  );
}
