// SPDX-License-Identifier: Apache-2.0
import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Button } from "./forms/Button.jsx";
import { InlineError } from "./feedback/InlineError.jsx";

describe("shared component states", () => {
  it("exposes loading state and blocks activation", () => {
    const onClick = vi.fn();
    render(<Button loading onClick={onClick}>Generate</Button>);
    const button = screen.getByRole("button", { name: "Generate" });
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("aria-busy", "true");
    fireEvent.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });

  it("announces server and validation errors", () => {
    const { rerender } = render(<InlineError />);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    rerender(<InlineError>Unable to generate</InlineError>);
    expect(screen.getByRole("alert")).toHaveTextContent("Unable to generate");
  });
});
