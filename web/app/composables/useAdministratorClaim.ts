export interface AdministratorClaimStatus {
  claimed: boolean;
  canClaim: boolean;
}

export function useAdministratorClaim() {
  const { call } = useApi();
  const status = useState<AdministratorClaimStatus | null>(
    "blog-administrator-claim-status",
    () => null,
  );
  const pending = useState("blog-administrator-claim-pending", () => false);

  async function refresh() {
    pending.value = true;
    try {
      status.value = await call<AdministratorClaimStatus>(
        "/api/v1/authorization/setup",
      );
      return status.value;
    } finally {
      pending.value = false;
    }
  }

  function markClaimed() {
    status.value = { claimed: true, canClaim: false };
  }

  return { status, pending, refresh, markClaimed };
}
