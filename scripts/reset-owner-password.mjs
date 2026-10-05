#!/usr/bin/env node
// Run only in a trusted operator environment; never expose the admin key to the frontend.
import { isIP } from "node:net";

async function main() {
  const required = [
    "SUPABASE_URL",
    "SUPABASE_SERVICE_ROLE_KEY",
    "OWNER_AUTH_USER_ID",
    "NEW_OWNER_PASSWORD",
  ];
  for (const name of required) {
    if (!process.env[name]?.trim()) {
      throw new Error(`${name} is required.`);
    }
  }

  const url = new URL(process.env.SUPABASE_URL);
  if (
    url.protocol !== "https:" ||
    url.username ||
    url.password ||
    url.search ||
    url.hash ||
    url.pathname !== "/" ||
    isIP(url.hostname)
  ) {
    throw new Error("SUPABASE_URL must be an HTTPS project URL without a path, credentials, query, or fragment.");
  }
  const userID = process.env.OWNER_AUTH_USER_ID;
  if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(userID)) {
    throw new Error("OWNER_AUTH_USER_ID must be a Supabase Auth user UUID.");
  }

  url.pathname = `/auth/v1/admin/users/${userID}`;
  const response = await fetch(url, {
    method: "PUT",
    headers: {
      apikey: process.env.SUPABASE_SERVICE_ROLE_KEY,
      Authorization: `Bearer ${process.env.SUPABASE_SERVICE_ROLE_KEY}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ password: process.env.NEW_OWNER_PASSWORD }),
    redirect: "error",
    signal: AbortSignal.timeout(30_000),
  });
  if (!response.ok) {
    // Do not print response bodies: auth responses can contain sensitive user data.
    throw new Error(`Supabase password update failed (HTTP ${response.status}). Check the project URL, admin key, user ID, and password policy.`);
  }
  console.log(`Password updated for Auth user ${userID}. Organization data was not changed.`);
}

main().catch((error) => {
  console.error(`Password reset failed: ${error.message}`);
  process.exitCode = 1;
});
