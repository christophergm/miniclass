import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { clearApplicationSession } from "@/lib/auth";
import { resourceApi } from "@/lib/apiResources";

export function GuardianLogoutButton({ className }: { className?: string }) {
  const navigate = useNavigate();
  const [isSigningOut, setIsSigningOut] = useState(false);

  async function signOut() {
    setIsSigningOut(true);
    try {
      await resourceApi.revokeAuthSession();
    } catch {
      // The local bearer must not survive a failed revoke request.
    } finally {
      clearApplicationSession();
      navigate("/guardian", { replace: true });
    }
  }

  return (
    <Button
      className={`cursor-pointer ${className ?? ""}`}
      disabled={isSigningOut}
      onClick={() => void signOut()}
      size="sm"
      type="button"
      variant="outline"
    >
      {isSigningOut ? "Signing out…" : "Sign out"}
    </Button>
  );
}
