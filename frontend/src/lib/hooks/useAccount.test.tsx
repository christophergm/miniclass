import { screen } from "@testing-library/react";
import { StrictMode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { resourceApi, type MeResponse } from "@/lib/apiResources";
import { renderWithQueryClient } from "@/test/queryClient";

import { useAccount, useAccountRole, useIsOwner } from "./useAccount";

const owner: MeResponse = {
  role: "owner",
  principal: { id: "user-test", email: "owner@example.test" },
  organization: { id: "org-test", name: "Synthetic Academy" },
};

// Three readers standing in for the three the application has: the shell reads
// the organisation, the audit log reads the role, and two pages gate an
// owner-only control on it.
function Shell() {
  const account = useAccount();
  return <p>organisation: {account.data?.organization.name ?? "…"}</p>;
}

function RoleReader() {
  return <p>role: {useAccountRole() ?? "…"}</p>;
}

function OwnerOnlyControl() {
  return useIsOwner() ? <p>owner control</p> : null;
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useAccount", () => {
  it("serves every reader of the signed-in account from one request", async () => {
    const getMe = vi.spyOn(resourceApi, "getMe").mockResolvedValue(owner);

    renderWithQueryClient(
      <StrictMode>
        <Shell />
        <RoleReader />
        <OwnerOnlyControl />
      </StrictMode>,
    );

    expect(await screen.findByText("owner control")).toBeInTheDocument();
    expect(screen.getByText("organisation: Synthetic Academy")).toBeInTheDocument();
    expect(screen.getByText("role: owner")).toBeInTheDocument();
    // One key, so three readers are three subscriptions to one cache entry.
    // Under the previous three keys this was three requests for one account.
    expect(getMe).toHaveBeenCalledTimes(1);
  });

  it("compares the role case-insensitively, so an owner keeps owner-only controls", async () => {
    vi.spyOn(resourceApi, "getMe").mockResolvedValue({ ...owner, role: "Owner" });

    renderWithQueryClient(
      <>
        <RoleReader />
        <OwnerOnlyControl />
      </>,
    );

    expect(await screen.findByText("owner control")).toBeInTheDocument();
    expect(screen.getByText("role: owner")).toBeInTheDocument();
  });

  it("offers no owner-only control to a coordinator", async () => {
    vi.spyOn(resourceApi, "getMe").mockResolvedValue({ ...owner, role: "coordinator" });

    renderWithQueryClient(
      <>
        <RoleReader />
        <OwnerOnlyControl />
      </>,
    );

    expect(await screen.findByText("role: coordinator")).toBeInTheDocument();
    expect(screen.queryByText("owner control")).not.toBeInTheDocument();
  });
});
