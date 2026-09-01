import { act, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  createLocalDevAuthClient,
  reportSessionEnded,
  type AuthClient,
  type DevTokenStatus,
} from "@/lib/auth";
import { AuthProvider } from "@/lib/hooks/AuthProvider";
import { useAuth } from "@/lib/hooks/useAuth";

import { SignInPage } from "./SignInPage";

// A configured client with no session: what the local fake-auth client looks
// like when VITE_DEV_TOKEN yields nothing usable.
function sessionlessClient(): AuthClient {
  return {
    auth: {
      getSession: vi.fn(async () => ({ data: { session: null }, error: null })),
      onAuthStateChange: vi.fn(() => ({ data: { subscription: { unsubscribe: vi.fn() } } })),
      signInWithPassword: vi.fn(async () => ({ data: { session: null, user: null }, error: null })),
      signOut: vi.fn(async () => ({ error: null })),
    },
  } as unknown as AuthClient;
}

function renderSignIn(props: { localDevAuth: boolean; devToken: DevTokenStatus }) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/sign-in"]}>
        <AuthProvider client={sessionlessClient()}>
          <SignInPage {...props} />
        </AuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function base64url(value: unknown): string {
  return btoa(JSON.stringify(value)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function token(payload: Record<string, unknown>): string {
  return `${base64url({ alg: "ES256", kid: "local", typ: "JWT" })}.${base64url(payload)}.c2lnbmF0dXJl`;
}

function SessionSurface({ devToken }: { devToken: DevTokenStatus }) {
  const { session } = useAuth();
  return session ? <p>Authenticated session</p> : <SignInPage localDevAuth devToken={devToken} />;
}

afterEach(() => {
  vi.useRealTimers();
});

describe("sign-in page under local development auth", () => {
  it("replaces the form with a banner naming make login when no token is set", async () => {
    renderSignIn({ localDevAuth: true, devToken: { kind: "missing" } });

    const banner = await screen.findByRole("alert");
    expect(banner).toHaveTextContent("Local development authentication has no token");
    expect(banner).toHaveTextContent("no VITE_DEV_TOKEN is set.");
    expect(banner).toHaveTextContent("make login");
    expect(banner).toHaveTextContent(/restart the Vite dev server/);

    expect(screen.queryByLabelText("Password")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Email")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Sign in" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Forgot password?" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "System health" })).toHaveAttribute("href", "/health");
  });

  it("says the token is unreadable when it cannot be decoded", async () => {
    renderSignIn({ localDevAuth: true, devToken: { kind: "unreadable" } });

    const banner = await screen.findByRole("alert");
    expect(banner).toHaveTextContent("VITE_DEV_TOKEN is not a readable token.");
    expect(banner).toHaveTextContent("make login");
    expect(screen.queryByLabelText("Password")).not.toBeInTheDocument();
  });

  it("names the expiry when the token has expired", async () => {
    renderSignIn({
      localDevAuth: true,
      devToken: { kind: "expired", expiresAt: new Date("2020-01-02T03:04:05Z") },
    });

    const banner = await screen.findByRole("alert");
    expect(banner).toHaveTextContent(/VITE_DEV_TOKEN expired on .*2020/);
    expect(banner).toHaveTextContent("make login");
    expect(screen.queryByLabelText("Password")).not.toBeInTheDocument();
  });

  it("renders the ordinary form when the dev token is usable", async () => {
    renderSignIn({
      localDevAuth: true,
      devToken: { kind: "valid", expiresAt: new Date("2099-01-01T00:00:00Z") },
    });

    expect(await screen.findByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Forgot password?" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("renders the ordinary form on the real Supabase path regardless of the dev token", async () => {
    renderSignIn({ localDevAuth: false, devToken: { kind: "missing" } });

    expect(await screen.findByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("changes the visible surface when a local session crosses its expiry boundary", async () => {
    vi.useFakeTimers();
    const now = new Date("2026-08-25T12:00:00Z");
    vi.setSystemTime(now);
    const expiresAt = new Date(now.getTime() + 1000);
    const localToken = token({ exp: expiresAt.getTime() / 1000 });
    const client = createLocalDevAuthClient(localToken, { kind: "valid", expiresAt }) as AuthClient;

    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <MemoryRouter initialEntries={["/sign-in"]}>
          <AuthProvider client={client}>
            <SessionSurface devToken={{ kind: "valid", expiresAt }} />
          </AuthProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    await act(async () => {});
    expect(screen.getByText("Authenticated session")).toBeInTheDocument();

    await act(async () => {
      vi.advanceTimersByTime(1000);
    });

    const banner = screen.getByRole("alert");
    expect(banner).toHaveTextContent(/VITE_DEV_TOKEN expired on .*2026/);
    expect(banner).toHaveTextContent("make login");
    expect(screen.queryByText("Authenticated session")).not.toBeInTheDocument();
  });

  it("explains an API-invalid session and signs out the persisted client", async () => {
    const client = sessionlessClient();
    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <MemoryRouter initialEntries={["/sign-in"]}>
          <AuthProvider client={client}>
            <SignInPage />
          </AuthProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    await act(async () => {});
    await act(async () => {
      reportSessionEnded({ kind: "api-invalid-token" });
    });

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Your session expired or is no longer valid. Please sign in again.",
    );
    expect(client.auth.signOut).toHaveBeenCalledTimes(1);
  });
});
