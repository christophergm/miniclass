import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { AuthChangeEvent, AuthClient, Session } from "@/lib/auth";

import { AuthProvider } from "./AuthProvider";
import { useAuth } from "./useAuth";

type AuthStateListener = (event: AuthChangeEvent, session: Session | null) => void;

function session(userId: string, accessToken = `${userId}-token`): Session {
  return {
    access_token: accessToken,
    expires_at: 4102444800,
    expires_in: 3600,
    refresh_token: `${userId}-refresh-token`,
    token_type: "bearer",
    user: { id: userId, email: `${userId}@example.test` },
  } as Session;
}

function authClient(initialSession: Session) {
  let listener: AuthStateListener | undefined;
  const client = {
    auth: {
      getSession: vi.fn(async () => ({ data: { session: initialSession }, error: null })),
      onAuthStateChange: vi.fn((nextListener: AuthStateListener) => {
        listener = nextListener;
        return { data: { subscription: { unsubscribe: vi.fn() } } };
      }),
      signOut: vi.fn(async () => ({ error: null })),
    },
  } as unknown as AuthClient;

  return {
    client,
    emit: (event: AuthChangeEvent, nextSession: Session | null) => listener?.(event, nextSession),
  };
}

function SessionProbe() {
  const { session: currentSession } = useAuth();
  return <span data-testid="session-user">{currentSession?.user.id ?? "signed-out"}</span>;
}

function renderProvider(client: AuthClient, queryClient: QueryClient) {
  return render(
    <QueryClientProvider client={queryClient}>
      <AuthProvider client={client}>
        <SessionProbe />
      </AuthProvider>
    </QueryClientProvider>,
  );
}

async function waitForInitialSession() {
  await waitFor(() => expect(screen.getByTestId("session-user")).toHaveTextContent("user-a"));
}

describe("AuthProvider query cache boundary", () => {
  it("clears resource caches when a different signed-in identity arrives", async () => {
    const harness = authClient(session("user-a"));
    const queryClient = new QueryClient();
    renderProvider(harness.client, queryClient);
    await waitForInitialSession();

    queryClient.setQueryData(["account"], { organization: { name: "A organisation" } });
    queryClient.setQueryData(["school-years", "year-a"], [{ id: "year-a", label: "A year" }]);

    act(() => harness.emit("SIGNED_OUT", null));
    expect(queryClient.getQueryData(["account"])).toEqual({
      organization: { name: "A organisation" },
    });

    act(() => harness.emit("SIGNED_IN", session("user-b")));

    expect(queryClient.getQueryData(["account"])).toBeUndefined();
    expect(queryClient.getQueryData(["school-years", "year-a"])).toBeUndefined();
    expect(screen.getByTestId("session-user")).toHaveTextContent("user-b");
  });

  it("keeps resource caches intact when the same identity refreshes its token", async () => {
    const harness = authClient(session("user-a"));
    const queryClient = new QueryClient();
    renderProvider(harness.client, queryClient);
    await waitForInitialSession();

    const account = { organization: { name: "A organisation" } };
    const schoolYears = [{ id: "year-a", label: "A year" }];
    queryClient.setQueryData(["account"], account);
    queryClient.setQueryData(["school-years", "year-a"], schoolYears);

    act(() => harness.emit("TOKEN_REFRESHED", session("user-a", "refreshed-token")));

    expect(queryClient.getQueryData(["account"])).toBe(account);
    expect(queryClient.getQueryData(["school-years", "year-a"])).toBe(schoolYears);
  });
});
