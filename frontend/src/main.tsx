import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import AnnotatePage from "./AnnotatePage.tsx";
import App from "./App.tsx";

const queryClient = new QueryClient();

// Two pages do not justify a router; Vite serves index.html for every path.
const Page = window.location.pathname === "/annotate" ? AnnotatePage : App;

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <Page />
    </QueryClientProvider>
  </StrictMode>,
);
