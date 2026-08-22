import {
  expect,
  type Browser,
  type BrowserContext,
  type BrowserContextOptions,
  type Page,
} from "@playwright/test";

export function requiredEnv(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

export async function loginE2E(
  browser: Browser,
  options: BrowserContextOptions = {},
  colorMode?: "light" | "dark",
  siteURL?: string,
): Promise<BrowserContext> {
  const context = await browser.newContext(options);
  if (colorMode) {
    await context.addInitScript((value) => {
      window.localStorage.setItem("nuxt-color-mode", value);
    }, colorMode);
  }
  const loginOrigin =
    process.env.BLOG_E2E_ACCOUNT_URL?.trim() ||
    requiredEnv("BLOG_E2E_IDENTITY_URL");
  const response = await context.request.post(
    new URL("/api/v1/auth/login", loginOrigin).toString(),
    {
      data: {
        email: requiredEnv("BLOG_E2E_EMAIL"),
        password: requiredEnv("BLOG_E2E_PASSWORD"),
      },
    },
  );
  expect(
    response.ok(),
    `identity login failed with HTTP ${response.status()}`,
  ).toBeTruthy();
  if (siteURL) {
    const sessionResponse = await context.request.get(
      new URL("/auth/login?return_to=/", siteURL).toString(),
    );
    expect(
      sessionResponse.ok(),
      `site session bootstrap failed with HTTP ${sessionResponse.status()}`,
    ).toBeTruthy();
  }
  return context;
}

export async function settleNuxt(page: Page) {
  await page.waitForLoadState("load", { timeout: 30_000 });
  await page.waitForLoadState("networkidle", { timeout: 30_000 });
  await page.waitForFunction(
    async () => {
      const root = document.querySelector("#__nuxt");
      if (!root || !("__vue_app__" in root)) return false;
      await document.fonts.ready;
      await new Promise<void>((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
      );
      const settledRoot = document.querySelector("#__nuxt");
      return Boolean(settledRoot && "__vue_app__" in settledRoot);
    },
    undefined,
    { timeout: 30_000 },
  );
}
