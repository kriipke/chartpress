import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./design/tokens.css"; // design-system tokens (colors, type, spacing, …)
import "./app/base.css"; // page surface + reset

import App from "./App";

const rootElement = document.getElementById("root");
const root = createRoot(rootElement);

root.render(
  <StrictMode>
    <App />
  </StrictMode>
);
