import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { productSites } from "./contracts";
import { loginE2E, requiredEnv, settleNuxt } from "./runtime";

const accountURL = requiredEnv("BLOG_E2E_ACCOUNT_URL");
const e2eEmail = requiredEnv("BLOG_E2E_EMAIL");
const e2ePassword = requiredEnv("BLOG_E2E_PASSWORD");
const missing = "__platform_e2e_missing__";
const imageFixtureBase64 =
  "iVBORw0KGgoAAAANSUhEUgAAAKAAAABaCAYAAAA/xl1SAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAAJcEhZcwAADsMAAA7DAcdvqGQAAAISSURBVHhe7dRRSkNRFENRJ+PIHH4HoVxQBEmhIG1ekh1Yv333yMa394/bJ+BCgLAiQFgRIKwIEFYECCsChBUBwooAYUWAsCJAWBEgrAgQVgQIKwKEFQHCigAf8J+p38MvArzjGVPfWUeAf7xi6rurCPCbY+oda+YDvMLUu1ZMB3ilqfctmAzwylPvbTYXYMLUu1tNBZg09f5GMwEmTt3RZiLA5Kl7mhDgxafuaVIfYMPUXS2qA2yauq8BAYZM3degNsDGqTvTEWDQ1J3pKgNsnro3GQGGTd2brC7Aham7UxFg4NTdqQgwcOruVFUBLk3dn4gAQ6fuT0SAoVP3JyLA0Kn7ExFg6NT9iQgwdOr+RAQYOnV/opoA16b+Bon4Dxg6dX8iAgyduj8RAYZO3Z+IAEOn7k9EgKFT9yciwNCp+xMRYOjU/YmqAjwWpu5ORYCBU3enIsDAqbtT1QV4NE/dm4wAw6buTVYZ4NE4dWc6AgyaujNdbYBH09R9DQgwZOq+BtUBHg1Td7WoD/BInrqnCQFefOqeJhMBHolTd7SZCfBImnp/o6kAj4Spd7eaC/C48tR7m00G+ONKU+9bMB3gcYWpd62YD/CHY+odawjwj1dMfXcVAd7xjKnvrCPAB/xn6vfwiwBhRYCwIkBYESCsCBBWBAgrAoQVAcKKAGFFgLAiQFgRIKwIEFYECCsChNHt8wtKVG37Suz0VQAAAABJRU5ErkJggg==";

function captureErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(`console: ${message.text()}`);
  });
  page.on("pageerror", (error) => errors.push(`pageerror: ${error.message}`));
  page.on("response", (response) => {
    if (response.status() >= 500)
      errors.push(`http ${response.status()}: ${response.url()}`);
  });
  return errors;
}

async function purgeTestPost(
  context: BrowserContext,
  siteURL: string,
  postId: string,
) {
  await context.request.delete(
    new URL(`/api/v1/posts/${postId}`, siteURL).toString(),
  );
  await context.request.delete(
    new URL(`/api/v1/posts/${postId}/permanent`, siteURL).toString(),
  );
}

export function registerJourneySuite(product: string) {
  for (const site of productSites(product)) {
    const contract = site.contract;
    test.describe(`${site.slug} (${site.product})`, () => {
      test("公开入口完成渲染且没有浏览器错误", async ({ page }) => {
        const errors = captureErrors(page);
        const response = await page.goto(
          new URL(contract.public.path, site.url).toString(),
          { waitUntil: "domcontentloaded" },
        );
        expect(response?.ok()).toBeTruthy();
        await expect(
          page.locator(contract.public.readySelector).first(),
        ).toBeVisible();
        await settleNuxt(page);
        expect(errors).toEqual([]);
      });

      test("首页精选封面加载后保持固定桌面构图", async ({ page }, testInfo) => {
        await page.setViewportSize({ width: 1440, height: 900 });
        let releaseMedia = () => undefined;
        const mediaGate = new Promise<void>((resolve) => {
          releaseMedia = resolve;
        });
        await page.route("**/media/**", async (route) => {
          await mediaGate;
          await route.continue();
        });
        const response = await page.goto(
          new URL(contract.public.path, site.url).toString(),
          { waitUntil: "domcontentloaded" },
        );
        expect(response?.ok()).toBeTruthy();

        const firstDot = page.getByRole("button", { name: "精选第 1 篇" });
        await expect(firstDot).toBeVisible();
        const featured = firstDot.locator("xpath=ancestor::section[1]");
        const lead = featured.locator('a[href^="/posts/"]').first();
        const image = lead.locator("img");
        await expect(image).toBeVisible();

        const measureFeatured = () =>
          page.evaluate(() => {
            const dot = document.querySelector<HTMLButtonElement>(
              'button[aria-label="精选第 1 篇"]',
            );
            const section = dot?.closest("section");
            const leadColumn = section?.firstElementChild;
            const card =
              section?.querySelector<HTMLAnchorElement>('a[href^="/posts/"]');
            const sectionBox = section?.getBoundingClientRect();
            const leadColumnBox = leadColumn?.getBoundingClientRect();
            const cardBox = card?.getBoundingClientRect();
            return {
              sectionHeight: sectionBox?.height ?? 0,
              sectionBottom: sectionBox?.bottom ?? 0,
              leadColumnHeight: leadColumnBox?.height ?? 0,
              cardHeight: cardBox?.height ?? 0,
              cardBottom: cardBox?.bottom ?? 0,
            };
          });

        const beforeLoad = await measureFeatured();
        releaseMedia();
        await image.evaluate((element: HTMLImageElement) => {
          if (element.complete && element.naturalWidth > 0) return;
          return new Promise<void>((resolve, reject) => {
            element.addEventListener("load", () => resolve(), { once: true });
            element.addEventListener(
              "error",
              () => reject(new Error("精选封面加载失败")),
              {
                once: true,
              },
            );
          });
        });
        const afterLoad = await measureFeatured();
        await page.screenshot({
          path: testInfo.outputPath("featured-cover-loaded.png"),
          fullPage: false,
        });

        for (const geometry of [beforeLoad, afterLoad]) {
          expect(geometry.sectionHeight).toBeLessThanOrEqual(452);
          expect(geometry.leadColumnHeight).toBeLessThanOrEqual(452);
          expect(geometry.cardHeight).toBeLessThanOrEqual(412);
          expect(geometry.cardBottom).toBeLessThanOrEqual(
            geometry.sectionBottom + 1,
          );
        }
        expect(
          Math.abs(afterLoad.cardHeight - beforeLoad.cardHeight),
        ).toBeLessThanOrEqual(1);
      });

      test("已发布文章在 LAN HTTP 下可以阅读", async ({ page }) => {
        const errors = captureErrors(page);
        await page.addInitScript(() => {
          Object.defineProperty(globalThis.crypto, "randomUUID", {
            configurable: true,
            value: undefined,
          });
        });
        await page.goto(new URL(contract.public.path, site.url).toString(), {
          waitUntil: "domcontentloaded",
        });
        const articleLink = page.locator('a[href^="/posts/"]').first();
        await expect(articleLink).toBeVisible();
        const articleTitle = (
          await articleLink.getByRole("heading").innerText()
        ).trim();
        const articlePath = await articleLink.getAttribute("href");
        expect(articlePath).toBeTruthy();

        await page.goto(new URL(articlePath!, site.url).toString(), {
          waitUntil: "domcontentloaded",
        });
        await page.waitForTimeout(1_500);
        expect(
          await page.getByText("crypto.randomUUID is not a function").count(),
        ).toBe(0);
        await expect(
          page.getByRole("heading", { name: articleTitle }).first(),
        ).toBeVisible();
        await settleNuxt(page);
        expect(errors).toEqual([]);
      });

      test("匿名访问管理入口进入账户登录流程", async ({ page }) => {
        const errors = captureErrors(page);
        await page.goto(new URL(contract.manage.path, site.url).toString(), {
          waitUntil: "domcontentloaded",
        });
        await expect(page).toHaveURL(
          new RegExp(
            `^${accountURL.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/login(?:\\?|$)`,
          ),
        );
        await expect(
          page.getByRole("heading", { name: "欢迎回来" }),
        ).toBeVisible();
        await settleNuxt(page);
        expect(errors).toEqual([]);
      });

      test("账户登录后阅读公开文章不会丢失管理会话", async ({ page }) => {
        const errors = captureErrors(page);
        const manageURL = new URL(contract.manage.path, site.url).toString();
        await page.goto(manageURL, { waitUntil: "domcontentloaded" });
        await expect(
          page.getByRole("heading", { name: "欢迎回来" }),
        ).toBeVisible();
        await settleNuxt(page);
        await page.getByRole("textbox", { name: /邮箱/ }).fill(e2eEmail);
        await page.getByLabel(/密码/).fill(e2ePassword);
        await page.getByRole("button", { name: "登录", exact: true }).click();
        await page.waitForURL(manageURL);
        await expect(page.locator("[data-blog-dashboard-panel]")).toBeVisible();

        await page.goto(new URL(contract.public.path, site.url).toString(), {
          waitUntil: "domcontentloaded",
        });
        const articlePath = await page
          .locator('a[href^="/posts/"]')
          .first()
          .getAttribute("href");
        expect(articlePath).toBeTruthy();
        await page.goto(new URL(articlePath!, site.url).toString(), {
          waitUntil: "domcontentloaded",
        });
        await page.waitForTimeout(1_500);
        await expect(
          page.getByRole("link", { name: "登录", exact: true }),
        ).toHaveCount(0);

        await page.goto(manageURL, { waitUntil: "domcontentloaded" });
        await expect(page).toHaveURL(manageURL);
        await expect(page.locator("[data-blog-dashboard-panel]")).toBeVisible();
        expect(errors).toEqual([]);
      });

      test("中央会话可以静默重建产品会话", async ({ page }) => {
        const manageURL = new URL(contract.manage.path, site.url).toString();
        await page.goto(manageURL, { waitUntil: "domcontentloaded" });
        await expect(
          page.getByRole("heading", { name: "欢迎回来" }),
        ).toBeVisible();
        await settleNuxt(page);
        await page.getByRole("textbox", { name: /邮箱/ }).fill(e2eEmail);
        await page.getByLabel(/密码/).fill(e2ePassword);
        await page.getByRole("button", { name: "登录", exact: true }).click();
        await page.waitForURL(manageURL);
        await expect(page.locator("[data-blog-dashboard-panel]")).toBeVisible();

        await page.context().clearCookies({ name: "rs_session" });
        await page
          .goto(manageURL, { waitUntil: "domcontentloaded" })
          .catch((error) => {
            if (!String(error).includes("ERR_ABORTED")) throw error;
          });
        await expect(page).toHaveURL(manageURL, { timeout: 10_000 });
        await expect(page.locator("[data-blog-dashboard-panel]")).toBeVisible();
        await expect(
          page.getByRole("heading", { name: "欢迎回来" }),
        ).toHaveCount(0);
      });

      test("已登录运营者可以进入管理界面", async ({ browser }) => {
        const context = await loginE2E(browser);
        const page = await context.newPage();
        const errors = captureErrors(page);
        try {
          const manageURL = new URL(contract.manage.path, site.url).toString();
          const manageResponse = await page.goto(manageURL, {
            waitUntil: "domcontentloaded",
          });
          const manageHTML = await manageResponse?.text();
          expect(manageHTML).not.toContain("正在打开控制台");
          expect(manageHTML).toContain("data-admin-shell");
          await expect(page).toHaveURL(manageURL);
          await expect(
            page.locator(contract.manage.readySelector).first(),
          ).toBeVisible();
          await expect(
            page
              .getByRole("heading", {
                name: new RegExp(contract.manage.heading),
              })
              .first(),
          ).toBeVisible();
          await expect(
            page.locator('[data-admin-sidebar-appearance="commercial"]'),
          ).toBeVisible();
          await expect(
            page.locator('[data-admin-sidebar-brand] a[href="/"]'),
          ).toBeVisible();
          await expect(
            page.locator("[data-admin-sidebar-brand]").getByText("月离博客", {
              exact: true,
            }),
          ).toBeVisible();
          await expect(
            page.getByText("控制台在线", { exact: true }),
          ).toHaveCount(0);
          await expect(
            page.locator("[data-blog-dashboard-panel]"),
          ).toBeVisible();
          await expect(page.locator("[data-dashboard-metric]")).toHaveCount(4);
          await expect(page.locator("[data-dashboard-trend]")).toBeVisible();
          await expect(page.locator("[data-dashboard-audience]")).toBeVisible();
          await expect(
            page.locator("[data-dashboard-top-posts]"),
          ).toBeVisible();
          await expect(page.locator("[data-dashboard-sources]")).toBeVisible();
          const sevenDayResponse = page.waitForResponse(
            (response) =>
              response.url().includes("/api/v1/dashboard/overview?days=7") &&
              response.ok(),
          );
          await page.getByRole("tab", { name: "7 天" }).click();
          await expect(page.locator("[data-dashboard-metric]")).toHaveCount(4);
          await sevenDayResponse;
          await expect(
            page.locator(
              '[data-dashboard-trend] [data-dashboard-trend-chart] svg[aria-label*="最近 7 天"]',
            ),
          ).toBeVisible();
          await expect(
            page.getByRole("heading", { name: "最近工作" }),
          ).toHaveCount(0);
          await expect(
            page.getByRole("heading", { name: "快捷动作" }),
          ).toHaveCount(0);
          await expect(
            page.getByRole("heading", { name: "待处理" }),
          ).toHaveCount(0);
          await expect(
            page.getByRole("button", { name: /打开.+站点菜单/ }),
          ).toHaveCount(0);

          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "domcontentloaded",
          });
          const postsHeader = page.locator("[data-manage-posts-header]");
          await expect(postsHeader).toBeVisible();
          await expect(
            postsHeader.getByRole("heading", { name: "文章", exact: true }),
          ).toBeVisible();
          await expect(
            postsHeader.getByRole("button", { name: "写新文章", exact: true }),
          ).toHaveCount(1);
          await expect(
            page.getByRole("navigation", { name: "内容状态" }),
          ).toHaveCount(0);

          await page.getByRole("button", { name: "筛选", exact: true }).click();
          await page.getByLabel("文章状态").click();
          await page.getByRole("option", { name: /已发布/ }).click();
          await expect(page).toHaveURL(/(?:\?|&)status=published(?:&|$)/);
          await expect(
            page.getByText("状态：已发布", { exact: true }),
          ).toBeVisible();

          await settleNuxt(page);
          expect(errors).toEqual([]);
        } finally {
          await context.close();
        }
      });

      test("管理页面使用统一页头且不重复解释页面职责", async ({ browser }) => {
        const context = await loginE2E(browser);
        const page = await context.newPage();
        const errors = captureErrors(page);
        const pages = [
          { path: "/manage", title: "控制台" },
          { path: "/manage/posts", title: "文章" },
          { path: "/manage/comments", title: "评论" },
          { path: "/manage/categories", title: "分类" },
          { path: "/manage/tags", title: "标签" },
          { path: "/manage/settings", title: "站点设置" },
          { path: "/manage/assets", title: "资源配置" },
          { path: "/manage/authorization", title: "权限与申请" },
        ];
        const headingStyles: Array<{ path: string; style: string }> = [];

        try {
          for (const target of pages) {
            const errorCount = errors.length;
            await page.goto(new URL(target.path, site.url).toString(), {
              waitUntil: "domcontentloaded",
            });
            const header = page.locator("[data-manage-page-header]");
            const heading = header.getByRole("heading", {
              name: target.title,
              exact: true,
            });
            await expect(header).toHaveCount(1);
            await expect(heading).toBeVisible();
            await page.evaluate(async () => {
              await document.fonts.ready;
              await new Promise<void>((resolve) =>
                requestAnimationFrame(() =>
                  requestAnimationFrame(() => resolve()),
                ),
              );
            });
            await expect(header.locator("p")).toHaveCount(0);
            await expect(page.locator("h1:visible")).toHaveCount(1);
            if (target.path === "/manage/comments") {
              await expect(
                page.getByText("全站评论审核", { exact: false }),
              ).toHaveCount(0);
            }
            if (target.path === "/manage/assets") {
              await expect(
                page.locator("[data-blog-asset-settings]"),
              ).toBeVisible();
              await expect(
                page.getByRole("heading", { name: "用途规则", exact: true }),
              ).toBeVisible();
            }
            headingStyles.push({
              path: target.path,
              style: await heading.evaluate((element) => {
                const style = getComputedStyle(element);
                return [
                  style.fontSize,
                  style.fontWeight,
                  style.lineHeight,
                ].join("/");
              }),
            });
            expect(
              errors.slice(errorCount),
              `${target.path} 不应产生浏览器错误`,
            ).toEqual([]);
          }

          expect(headingStyles).toEqual(
            headingStyles.map((item) => ({
              path: item.path,
              style: headingStyles[0]!.style,
            })),
          );
        } finally {
          await context.close();
        }
      });

      test("管理主题使用博客蓝并支持明暗模式", async ({ browser }) => {
        for (const colorMode of ["light", "dark"] as const) {
          const context = await loginE2E(
            browser,
            { viewport: { width: 1280, height: 600 } },
            colorMode,
          );
          const page = await context.newPage();
          const errors = captureErrors(page);
          try {
            await page.goto(new URL("/manage/comments", site.url).toString(), {
              waitUntil: "domcontentloaded",
            });
            await expect(
              page.getByRole("heading", { name: "评论", exact: true }),
            ).toBeVisible();
            const activeIcon = page.locator(
              '[data-admin-sidebar-primary] [data-slot="link"][data-active] [data-slot="linkLeadingIcon"]',
            );
            await expect(activeIcon).toBeVisible();
            const theme = await page.evaluate(() => {
              const bodyStyle = getComputedStyle(document.body);
              const activeIcon = document.querySelector(
                '[data-admin-sidebar-primary] [data-slot="link"][data-active] [data-slot="linkLeadingIcon"]',
              );
              const inactiveIcon = document.querySelector(
                '[data-admin-sidebar-primary] [data-slot="link"]:not([data-active]) [data-slot="linkLeadingIcon"]',
              );
              const activeIconStyle = activeIcon
                ? getComputedStyle(activeIcon)
                : null;
              const inactiveIconStyle = inactiveIcon
                ? getComputedStyle(inactiveIcon)
                : null;
              const sidebar = document.querySelector(
                '[data-admin-sidebar-appearance="commercial"]',
              );
              const sidebarBody = sidebar?.querySelector('[data-slot="body"]');
              const sidebarFooter = sidebar?.querySelector(
                "[data-admin-sidebar-account]",
              );
              const footerRect = sidebarFooter?.getBoundingClientRect();
              return {
                dark: document.documentElement.classList.contains("dark"),
                body: bodyStyle.backgroundColor,
                activeIcon: activeIconStyle?.color,
                activeIconPaint: activeIconStyle?.backgroundColor,
                activeIconOpacity: activeIconStyle?.opacity,
                inactiveIcon: inactiveIconStyle?.color,
                inactiveIconPaint: inactiveIconStyle?.backgroundColor,
                inactiveIconOpacity: inactiveIconStyle?.opacity,
                sidebarHeight: sidebar?.getBoundingClientRect().height,
                viewportHeight: window.innerHeight,
                navigationCount: sidebar?.querySelectorAll(
                  '[data-admin-sidebar-primary] [data-slot="link"]',
                ).length,
                sidebarOverflowY: sidebarBody
                  ? getComputedStyle(sidebarBody).overflowY
                  : "",
                footerVisible: Boolean(
                  footerRect &&
                  footerRect.top >= 0 &&
                  footerRect.bottom <= window.innerHeight,
                ),
              };
            });
            expect({
              dark: theme.dark,
              body: theme.body,
              activeIcon: theme.activeIcon,
              activeIconPaint: theme.activeIconPaint,
              activeIconOpacity: theme.activeIconOpacity,
              inactiveIcon: theme.inactiveIcon,
              inactiveIconPaint: theme.inactiveIconPaint,
              inactiveIconOpacity: theme.inactiveIconOpacity,
            }).toEqual(
              colorMode === "dark"
                ? {
                    dark: true,
                    body: "rgb(11, 15, 23)",
                    activeIcon: "rgb(147, 197, 253)",
                    activeIconPaint: "rgb(147, 197, 253)",
                    activeIconOpacity: "1",
                    inactiveIcon: "rgb(168, 183, 202)",
                    inactiveIconPaint: "rgb(168, 183, 202)",
                    inactiveIconOpacity: "1",
                  }
                : {
                    dark: false,
                    body: "rgb(248, 250, 252)",
                    activeIcon: "rgb(37, 99, 235)",
                    activeIconPaint: "rgb(37, 99, 235)",
                    activeIconOpacity: "1",
                    inactiveIcon: "rgb(71, 85, 105)",
                    inactiveIconPaint: "rgb(71, 85, 105)",
                    inactiveIconOpacity: "1",
                  },
            );
            expect(theme.sidebarHeight).toBe(theme.viewportHeight);
            expect(theme.navigationCount).toBe(8);
            expect(theme.sidebarOverflowY).toBe("auto");
            expect(theme.footerVisible).toBe(true);
            await expect(page.getByText(/控制台在线|博客服务在线/)).toHaveCount(
              0,
            );
            expect(errors).toEqual([]);
          } finally {
            await context.close();
          }
        }
      });

      test("文章列表快捷入口与编辑工作台可用", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "domcontentloaded",
          });
          const publicLink = page
            .getByRole("link", { name: /查看前台文章/ })
            .first();
          await expect(publicLink).toBeVisible();
          await expect(publicLink).toHaveAttribute("href", /^\/posts\//);
          await expect(publicLink).toHaveAttribute("target", "_blank");
          const publicLabel = await publicLink.getAttribute("aria-label");
          const publishedTitle =
            publicLabel?.replace(/^查看前台文章：/, "") || "";
          expect(publishedTitle).not.toBe("");
          const editLink = page.getByRole("link", {
            name: `编辑文章：${publishedTitle}`,
          });
          const editHref = await editLink.getAttribute("href");
          expect(editHref).toMatch(/^\/manage\/posts\//);

          await page.goto(new URL(editHref!, site.url).toString(), {
            waitUntil: "domcontentloaded",
          });
          await expect(page.locator("[data-blog-post-editor]")).toBeVisible();
          await expect(
            page.locator("[data-blog-editor-commandbar]"),
          ).toBeVisible();
          await expect(
            page.locator("[data-blog-editor-document]"),
          ).toBeVisible();
          await expect(page.getByLabel("文章标题")).toBeVisible();
          const documentBox = await page
            .locator("[data-blog-editor-document]")
            .boundingBox();
          const articleBox = await page
            .locator("[data-blog-editor-document] .ProseMirror")
            .boundingBox();
          expect(documentBox?.width || 0).toBeGreaterThan(1_050);
          expect(articleBox?.width || 0).toBeGreaterThan(950);
          const editorPresentation = await page.evaluate(() => {
            const documentElement = document.querySelector(
              "[data-blog-editor-document]",
            );
            const title = document.querySelector(".blog-editor-title");
            return {
              background: documentElement
                ? getComputedStyle(documentElement).backgroundColor
                : "",
              titleFontSize: title
                ? Number.parseFloat(getComputedStyle(title).fontSize)
                : 0,
            };
          });
          expect(editorPresentation.background).not.toBe("rgba(0, 0, 0, 0)");
          expect(editorPresentation.titleFontSize).toBeLessThanOrEqual(36);
          await expect(
            page.getByRole("link", { name: "查看前台文章" }),
          ).toHaveAttribute("href", /^\/posts\//);
          await page.getByRole("button", { name: "文章设置" }).click();
          await expect(
            page.getByRole("heading", { name: "文章设置", exact: true }),
          ).toBeVisible();
          await expect(
            page.locator("[data-blog-settings-accordion]"),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: /封面与摘要/ }),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: /分类与系列/ }),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: /发布设置/ }),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: /搜索优化/ }),
          ).toBeVisible();
          const inspectorPresentation = await page.evaluate(() => {
            const surface = document.querySelector(
              ".blog-editor-settings-surface",
            );
            const accordion = document.querySelector(
              "[data-blog-settings-accordion]",
            );
            const section = document.querySelector(
              ".blog-editor-settings-section",
            );
            const styles = surface ? getComputedStyle(surface) : null;
            return {
              background: styles?.getPropertyValue("--ui-bg").trim() || "",
              muted: styles?.getPropertyValue("--ui-bg-muted").trim() || "",
              sectionShadow: section ? getComputedStyle(section).boxShadow : "",
              radius: accordion
                ? Number.parseFloat(getComputedStyle(accordion).borderRadius)
                : 0,
              decorativeIcons: document.querySelectorAll(
                "[data-blog-settings-accordion] [data-slot='trigger'] > [data-slot='leadingIcon']",
              ).length,
            };
          });
          expect(inspectorPresentation.background).toBe("#ffffff");
          expect(inspectorPresentation.muted).toBe("#f6f8fa");
          expect(inspectorPresentation.sectionShadow).toBe("none");
          expect(inspectorPresentation.radius).toBeLessThanOrEqual(10);
          expect(inspectorPresentation.decorativeIcons).toBe(0);
          const coverBox = await page
            .locator("[data-blog-cover-preview]")
            .boundingBox();
          expect(coverBox?.height || 0).toBeLessThanOrEqual(90);
          expect((coverBox?.width || 0) / (coverBox?.height || 1)).toBeCloseTo(
            3 / 2,
            1,
          );
          await page.getByRole("button", { name: /发布设置/ }).click();
          await expect(
            page.getByText("当前状态", { exact: true }),
          ).toBeVisible();
          await page.getByRole("button", { name: /搜索优化/ }).click();
          await expect(page.getByLabel("Meta 标题")).toBeVisible();
          await page
            .getByRole("button", { name: "保存", exact: true })
            .last()
            .click();
          await expect(
            page.getByRole("button", { name: "已保存", exact: true }).last(),
          ).toBeVisible();

          const overflow = await page.evaluate(() => ({
            viewport: document.documentElement.clientWidth,
            document: document.documentElement.scrollWidth,
          }));
          expect(overflow.document).toBeLessThanOrEqual(overflow.viewport);

          const layers = await page.evaluate(async () => {
            const sidebar = document.querySelector(
              '[data-admin-sidebar-appearance="commercial"]',
            );
            const account = sidebar?.querySelector(
              "[data-admin-sidebar-account]",
            );
            const main = document.querySelector("#manage-main");
            const commandbar = document.querySelector(
              "[data-blog-editor-commandbar]",
            );
            if (!sidebar || !account || !main || !commandbar)
              throw new Error("管理层级缺失");
            main.scrollTop = main.scrollHeight;
            await new Promise<void>((resolve) =>
              requestAnimationFrame(() =>
                requestAnimationFrame(() => resolve()),
              ),
            );
            const sidebarRect = sidebar.getBoundingClientRect();
            const accountRect = account.getBoundingClientRect();
            const mainRect = main.getBoundingClientRect();
            const commandbarRect = commandbar.getBoundingClientRect();
            return {
              viewportHeight: window.innerHeight,
              windowScrollY: window.scrollY,
              sidebarTop: sidebarRect.top,
              sidebarBottom: sidebarRect.bottom,
              accountBottom: accountRect.bottom,
              mainTop: mainRect.top,
              mainBottom: mainRect.bottom,
              commandbarTop: commandbarRect.top,
              mainScrollTop: main.scrollTop,
              mainClientHeight: main.clientHeight,
              mainScrollHeight: main.scrollHeight,
            };
          });
          expect(layers.windowScrollY).toBe(0);
          expect(layers.sidebarTop).toBe(0);
          expect(layers.sidebarBottom).toBe(layers.viewportHeight);
          expect(layers.accountBottom).toBeLessThanOrEqual(
            layers.viewportHeight,
          );
          expect(layers.mainTop).toBe(0);
          expect(layers.commandbarTop).toBe(0);
          expect(layers.mainBottom).toBe(layers.viewportHeight);
          expect(layers.mainScrollTop).toBeGreaterThan(0);
          expect(layers.mainScrollHeight).toBeGreaterThan(
            layers.mainClientHeight,
          );
          expect(errors).toEqual([]);
        } finally {
          await context.close();
        }
      });

      test("编辑器删除直接进入回收站并可恢复", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        let postId = "";
        let slug = "";
        let nativeDialogs = 0;
        page.on("dialog", async (dialog) => {
          nativeDialogs += 1;
          await dialog.dismiss();
        });
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `回收站验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          slug = created.post.slug;
          const publish = await context.request.patch(
            new URL(`/api/v1/posts/${postId}`, site.url).toString(),
            { data: { status: "published" } },
          );
          expect(publish.ok()).toBeTruthy();

          await page.goto(
            new URL(`/manage/posts/${slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await expect(
            page.getByText("文章编辑", { exact: true }),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: "更多文章操作" }),
          ).toHaveCount(0);
          await page.getByRole("button", { name: "文章设置" }).click();
          const settings = page.getByRole("dialog", { name: "文章设置" });
          const trashButton = settings.getByRole("button", {
            name: "移入回收站",
          });
          const settingsSave = settings.getByRole("button", {
            name: "保存",
            exact: true,
          });
          const trashBox = await trashButton.boundingBox();
          const saveBox = await settingsSave.boundingBox();
          expect(trashBox?.x || 0).toBeLessThan(saveBox?.x || 0);

          const trashed = page.waitForResponse(
            (response) =>
              response.request().method() === "DELETE" &&
              response.url().endsWith(`/api/v1/posts/${postId}`),
          );
          await trashButton.click();
          expect((await trashed).ok()).toBeTruthy();
          await expect(page).toHaveURL(/\/manage\/posts\?status=trash$/);
          expect(nativeDialogs).toBe(0);
          await expect(
            page.getByText(created.post.title, { exact: true }),
          ).toBeVisible();

          const hidden = await context.request.get(
            new URL(`/api/v1/posts/${slug}`, site.url).toString(),
          );
          expect(hidden.status()).toBe(404);

          const restored = page.waitForResponse(
            (response) =>
              response.request().method() === "POST" &&
              response.url().endsWith(`/api/v1/posts/${postId}/restore`),
          );
          await page
            .getByRole("button", { name: `恢复文章：${created.post.title}` })
            .click();
          expect((await restored).ok()).toBeTruthy();
          await expect(
            page.getByText(created.post.title, { exact: true }),
          ).toHaveCount(0);
          const visible = await context.request.get(
            new URL(`/api/v1/posts/${slug}`, site.url).toString(),
          );
          expect(visible.ok()).toBeTruthy();
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("文章发布日期与搜索信息可以维护", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const title = `文章元数据验收 ${Date.now()}`;
          const excerpt = "这是用于搜索结果和文章卡片的摘要。";
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title,
                content: "<h2>正文标题</h2><p>正文内容。</p>",
                excerpt,
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          const slug = created.post.slug;
          const publish = await context.request.patch(
            new URL(`/api/v1/posts/${postId}`, site.url).toString(),
            { data: { status: "published" } },
          );
          expect(publish.ok()).toBeTruthy();

          await page.goto(
            new URL(`/manage/posts/${slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await page.getByRole("button", { name: "文章设置" }).click();
          const settings = page.getByRole("dialog", { name: "文章设置" });
          await settings.getByRole("button", { name: /发布设置/ }).click();
          await expect(
            settings.getByText("展示浏览量", { exact: true }),
          ).toHaveCount(0);
          await expect(
            settings.getByText("修改后会影响归档顺序和文章显示日期。", {
              exact: true,
            }),
          ).toHaveCount(0);
          const localPublishedAt = "2026-07-01T08:30";
          await settings.getByLabel("发布日期").fill(localPublishedAt);

          await settings.getByRole("button", { name: /搜索优化/ }).click();
          await settings.getByRole("button", { name: "从文章填充" }).click();
          await expect(settings.getByLabel("Meta 标题")).toHaveValue(title);
          await expect(settings.getByLabel("OG 标题")).toHaveValue(title);
          await expect(settings.getByLabel("Meta 描述")).toHaveValue(excerpt);

          const metadataResponses = Promise.all([
            page.waitForResponse(
              (response) =>
                response.request().method() === "PATCH" &&
                response.url().endsWith(`/api/v1/posts/${postId}`),
            ),
            page.waitForResponse(
              (response) =>
                response.request().method() === "PUT" &&
                response.url().endsWith(`/api/v1/posts/${postId}/seo`),
            ),
          ]);
          await settings
            .getByRole("button", { name: "保存", exact: true })
            .click();
          for (const response of await metadataResponses)
            expect(response.ok()).toBeTruthy();

          const detail = await context.request.get(
            new URL(`/api/v1/posts/${slug}`, site.url).toString(),
          );
          expect(detail.ok()).toBeTruthy();
          const detailBody = await detail.json();
          expect(new Date(detailBody.post.publishedAt).toISOString()).toBe(
            new Date(localPublishedAt).toISOString(),
          );
          expect(detailBody.seo.metaTitle).toBe(title);
          expect(detailBody.seo.metaDesc).toBe(excerpt);
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("保存结果原地反馈且错误 Toast 保持低干扰", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        let blockerId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const blocker = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `保存冲突占位 ${Date.now()}`,
                content: "<p>blocker</p>",
              },
            },
          );
          expect(blocker.ok()).toBeTruthy();
          const blocked = await blocker.json();
          blockerId = blocked.post.id;
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `保存反馈验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          const originalSlug = created.post.slug;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );

          const postRoute = `**/api/v1/posts/${postId}`;
          await page.route(postRoute, async (route) => {
            if (route.request().method() !== "PATCH") {
              await route.continue();
              return;
            }
            await new Promise((resolve) => setTimeout(resolve, 250));
            await route.continue();
          });

          await page.getByLabel("文章永久链接").fill(blocked.post.slug);
          await page
            .getByRole("button", { name: "保存", exact: true })
            .last()
            .click();
          await expect(
            page.getByRole("button", { name: "保存中", exact: true }).last(),
          ).toBeVisible();
          const errorToast = page
            .locator("[data-yueli-toast]")
            .filter({ hasText: "保存失败" })
            .last();
          await expect(errorToast).toBeVisible({ timeout: 8_000 });
          await expect(
            errorToast.locator("[data-yueli-toast-progress]"),
          ).toHaveCount(0);
          await expect(
            errorToast.getByRole("button", { name: "关闭通知" }),
          ).toBeVisible();
          const presentation = await errorToast.evaluate((element) => {
            const style = getComputedStyle(element);
            const box = element.getBoundingClientRect();
            return {
              width: box.width,
              bottom: window.innerHeight - box.bottom,
              right: window.innerWidth - box.right,
              borderWidth: style.borderTopWidth,
              radius: Number.parseFloat(style.borderRadius),
              shadow: style.boxShadow,
              background: style.backgroundColor,
            };
          });
          expect(presentation.width).toBeLessThanOrEqual(352);
          expect(presentation.bottom).toBeLessThanOrEqual(32);
          expect(presentation.right).toBeLessThanOrEqual(32);
          expect(presentation.borderWidth).toBe("1px");
          expect(presentation.radius).toBeGreaterThanOrEqual(12);
          expect(presentation.shadow).not.toBe("none");
          expect(presentation.background).not.toBe("rgba(0, 0, 0, 0)");
          await expect(
            errorToast.getByText("文章地址已被占用，请换一个地址后重试。", {
              exact: true,
            }),
          ).toBeVisible();

          await page.unroute(postRoute);
          await errorToast.getByRole("button", { name: "关闭通知" }).click();
          await expect(errorToast).toHaveCount(0);
          await page.getByLabel("文章永久链接").fill(originalSlug);
          const saved = page.waitForResponse(
            (response) =>
              response.request().method() === "PATCH" &&
              response.url().endsWith(`/api/v1/posts/${postId}`),
          );
          await page
            .getByRole("button", { name: "保存", exact: true })
            .last()
            .click();
          expect((await saved).ok()).toBeTruthy();
          await expect(
            page.getByRole("button", { name: "已保存", exact: true }).last(),
          ).toBeVisible();
          await expect(
            page
              .locator("[data-yueli-toast]")
              .getByText("已保存", { exact: true }),
          ).toHaveCount(0);
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          if (blockerId) await purgeTestPost(context, site.url, blockerId);
          await context.close();
        }
      });

      test("代码与公式块支持源码预览和重新编辑", async ({
        browser,
      }, testInfo) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        let postId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `富文本块验收 ${Date.now()}`,
                content:
                  "```javascript\nconst answer = 1\n```\n\n$$\nE = mc^2\n$$",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          const editorURL = new URL(
            `/manage/posts/${created.post.slug}`,
            site.url,
          ).toString();
          await page.goto(editorURL, { waitUntil: "networkidle" });
          await settleNuxt(page);
          expect(errors).toEqual([]);

          const codeBlock = page.locator("[data-editor-code-block]");
          const mathBlock = page.locator("[data-editor-math-block]");
          await expect(codeBlock).toBeVisible();
          await expect(mathBlock).toBeVisible();
          await codeBlock.getByRole("button", { name: "编辑代码" }).click();
          await page.waitForTimeout(250);
          expect(errors).toEqual([]);
          await expect(
            codeBlock.locator("[data-editor-code-source]"),
          ).toBeVisible();
          await expect(
            codeBlock.locator("[data-editor-code-preview]"),
          ).toBeVisible();
          await codeBlock.locator("textarea").fill("const answer = 2");
          await codeBlock.getByRole("button", { name: "完成代码" }).click();

          await mathBlock.getByRole("button", { name: "编辑公式" }).click();
          await expect(
            mathBlock.locator("[data-editor-math-source]"),
          ).toBeVisible();
          await expect(
            mathBlock.locator("[data-editor-math-preview]"),
          ).toBeVisible();
          await page.screenshot({
            path: testInfo.outputPath("rich-block-editing.png"),
            fullPage: true,
          });
          await mathBlock.locator("textarea").fill("E = mc^3");
          await mathBlock.getByRole("button", { name: "完成公式" }).click();

          const editable = page.locator('.tiptap[contenteditable="true"]');
          await editable.click();
          await page.keyboard.press("Control+End");
          await page.keyboard.press("Enter");
          await page.keyboard.type("/");
          const inlineMathItem = page
            .getByText("行内公式", { exact: true })
            .last();
          await expect(inlineMathItem).toBeVisible();
          await inlineMathItem.click();
          await expect(
            editable.locator(
              'span[data-type="inline-math"][data-latex="E = mc^2"]',
            ),
          ).toBeVisible();
          await expect(editable).not.toContainText("$E=mc^2$");

          const saved = page.waitForResponse(
            (response) =>
              response.request().method() === "PATCH" &&
              response.url().endsWith(`/api/v1/posts/${postId}`),
          );
          await page
            .getByRole("button", { name: "保存", exact: true })
            .last()
            .click();
          expect((await saved).ok()).toBeTruthy();
          await expect(
            page.getByRole("button", { name: "已保存", exact: true }).last(),
          ).toBeVisible();

          await page.reload({ waitUntil: "networkidle" });
          const reopenedCode = page.locator("[data-editor-code-block]");
          const reopenedMath = page.locator("[data-editor-math-block]");
          await expect(
            page.locator(
              '.tiptap span[data-type="inline-math"][data-latex="E = mc^2"]',
            ),
          ).toBeVisible();
          await reopenedCode.locator("[data-editor-code-preview]").click();
          await reopenedMath.locator("[data-editor-math-preview]").click();
          await expect(reopenedCode.locator("textarea")).toHaveValue(
            "const answer = 2",
          );
          await expect(reopenedMath.locator("textarea")).toHaveValue(
            "E = mc^3",
          );
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("封面上传通过站点同源 Asset 代理", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        let postId = "";
        let coverAssetId = "";
        let inlineAssetId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `Asset 地址验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(
            create.ok(),
            `create temporary post failed: ${create.status()}`,
          ).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          const slug = created.post.slug;

          await page.goto(
            new URL(`/manage/posts/${slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await page.getByRole("button", { name: "文章设置" }).click();
          await expect(
            page.locator("[data-blog-settings-accordion]"),
          ).toBeVisible();

          const uploadFlow = Promise.all([
            page.waitForResponse(
              (response) =>
                response.request().method() === "PUT" &&
                response.url().includes("/asset-api/api/v1/assets/blob/"),
            ),
            page.waitForResponse(
              (response) =>
                response.request().method() === "POST" &&
                response
                  .url()
                  .includes(`/api/v1/posts/${postId}/cover/finalize`),
            ),
          ]);
          const coverInput = page.locator(
            '.blog-editor-settings input[type="file"][accept="image/*"]',
          );
          const realUploadFixture = process.env.BLOG_E2E_UPLOAD_FILE?.trim();
          let fixtureBuffer = realUploadFixture
            ? await readFile(realUploadFixture)
            : Buffer.from(imageFixtureBase64, "base64");
          const paddedBytes = Number(
            process.env.BLOG_E2E_UPLOAD_PAD_BYTES || "0",
          );
          if (
            Number.isSafeInteger(paddedBytes) &&
            paddedBytes > fixtureBuffer.length
          ) {
            fixtureBuffer = Buffer.concat([
              fixtureBuffer,
              Buffer.alloc(paddedBytes - fixtureBuffer.length),
            ]);
          }
          await coverInput.setInputFiles({
            name: `cover-probe-${Date.now()}.png`,
            mimeType: "image/png",
            buffer: fixtureBuffer,
          });
          await expect(
            page.locator("[data-asset-image-processor-controls]"),
          ).toBeVisible();
          await page.getByRole("button", { name: "使用处理后的图片" }).click();
          const [upload, finalize] = await uploadFlow;
          expect(upload.status()).toBe(200);
          expect(finalize.status()).toBe(200);
          expect(upload.url()).toMatch(
            new RegExp(
              `^${site.url.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/asset-api/`,
            ),
          );
          await expect(page.locator('img[alt="文章封面"]')).toBeVisible();

          const detail = await context.request.get(
            new URL(`/api/v1/posts/${slug}`, site.url).toString(),
          );
          expect(detail.ok()).toBeTruthy();
          const detailBody = await detail.json();
          coverAssetId = detailBody.post.coverAssetId || "";
          expect(detailBody.post.coverUrl).toMatch(
            new RegExp(
              `^${site.url.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/asset-api/`,
            ),
          );

          await page.keyboard.press("Escape");
          const inlineName = `inline-probe-${Date.now()}.png`;
          const inlineProcessedName = inlineName.replace(
            /\.png$/,
            "-crop.webp",
          );
          const inlineFileChooser = page.waitForEvent("filechooser");
          await page
            .locator(".blog-editor-rich-text button:has(.i-tabler\\:photo)")
            .first()
            .click();
          await (
            await inlineFileChooser
          ).setFiles({
            name: inlineName,
            mimeType: "image/png",
            buffer: fixtureBuffer,
          });
          await expect(
            page.getByRole("heading", { name: "处理正文图片" }),
          ).toBeVisible();
          await expect(page.getByText("最终预览", { exact: true })).toHaveCount(
            0,
          );
          const imageToolbar = page.getByRole("toolbar", {
            name: "图片变换工具",
          });
          await expect(imageToolbar).toBeVisible();
          await expect(
            imageToolbar.getByRole("button", { name: "向左旋转" }),
          ).toBeVisible();
          await expect(
            imageToolbar.getByRole("button", { name: "向右旋转" }),
          ).toBeVisible();
          const stageBox = await page
            .locator(".asset-image-cropper-stage")
            .boundingBox();
          const toolbarBox = await imageToolbar.boundingBox();
          expect(toolbarBox?.y || 0).toBeGreaterThanOrEqual(
            (stageBox?.y || 0) + (stageBox?.height || 0),
          );
          const editorHelp = page.getByRole("button", { name: "图片编辑帮助" });
          await expect(editorHelp).toBeVisible();
          await expect(
            page.getByText(
              "拖动图片调整位置，滚轮或工具栏缩放，拖动裁剪框调整范围。",
              { exact: true },
            ),
          ).toHaveCount(0);
          await editorHelp.hover();
          await expect(
            page
              .getByText(
                "拖动图片调整位置；滚轮或工具栏缩放；拖动裁剪框调整范围。",
                { exact: true },
              )
              .first(),
          ).toBeVisible();
          await page.mouse.move(0, 0);
          const zoomSlider = imageToolbar.getByRole("slider", {
            name: "缩放比例",
          });
          await expect(zoomSlider).toBeVisible();
          await expect(zoomSlider).toHaveValue("100");
          await expect(
            imageToolbar.getByText("100%", { exact: true }),
          ).toBeVisible();
          const freeCrop = await page.evaluate(() => {
            const image = document.querySelector(
              ".asset-image-cropper-stage img.cropper-hidden",
            ) as
              | (HTMLImageElement & {
                  cropper?: {
                    getData: (rounded?: boolean) => {
                      x: number;
                      y: number;
                      width: number;
                      height: number;
                    };
                    getImageData: () => {
                      naturalWidth: number;
                      naturalHeight: number;
                    };
                  };
                })
              | null;
            const cropper = image?.cropper;
            if (!cropper) throw new Error("正文图片裁剪器未就绪");
            return {
              crop: cropper.getData(true),
              image: cropper.getImageData(),
            };
          });
          expect(freeCrop.crop.x).toBeCloseTo(0, 0);
          expect(freeCrop.crop.y).toBeCloseTo(0, 0);
          expect(freeCrop.crop.width).toBeCloseTo(
            freeCrop.image.naturalWidth,
            0,
          );
          expect(freeCrop.crop.height).toBeCloseTo(
            freeCrop.image.naturalHeight,
            0,
          );
          const canvasSize = () =>
            page.evaluate(() => {
              const image = document.querySelector(
                ".asset-image-cropper-stage img.cropper-hidden",
              ) as
                | (HTMLImageElement & {
                    cropper?: {
                      getCanvasData: () => { width: number; height: number };
                    };
                  })
                | null;
              const cropper = image?.cropper;
              if (!cropper) throw new Error("正文图片裁剪器未就绪");
              const canvas = cropper.getCanvasData();
              return { width: canvas.width, height: canvas.height };
            });
          const beforeZoom = await canvasSize();
          await imageToolbar.getByRole("button", { name: "放大" }).click();
          const afterZoom = await canvasSize();
          expect(afterZoom.width).toBeGreaterThan(beforeZoom.width);
          expect(afterZoom.height).toBeGreaterThan(beforeZoom.height);
          await expect(zoomSlider).toHaveValue("110");
          await expect(
            imageToolbar.getByText("110%", { exact: true }),
          ).toBeVisible();
          await zoomSlider.fill("220");
          await expect(zoomSlider).toHaveValue("220");
          const zoomedCanvas = await page.evaluate(() => {
            const image = document.querySelector(
              ".asset-image-cropper-stage img.cropper-hidden",
            ) as
              | (HTMLImageElement & {
                  cropper?: {
                    getCanvasData: () => { width: number; height: number };
                  };
                })
              | null;
            const canvas = document.querySelector(
              ".asset-image-cropper-stage .cropper-canvas",
            );
            const stage = document.querySelector(".asset-image-cropper-stage");
            if (!image?.cropper || !canvas || !stage) {
              throw new Error("缩放画布尚未准备好");
            }
            return {
              modelWidth: image.cropper.getCanvasData().width,
              renderedWidth: canvas.getBoundingClientRect().width,
              stageWidth: stage.getBoundingClientRect().width,
            };
          });
          expect(zoomedCanvas.modelWidth).toBeGreaterThan(
            zoomedCanvas.stageWidth,
          );
          expect(zoomedCanvas.renderedWidth).toBeCloseTo(
            zoomedCanvas.modelWidth,
            0,
          );
          await imageToolbar.getByRole("button", { name: "水平翻转" }).click();
          await imageToolbar.getByRole("button", { name: "垂直翻转" }).click();
          const flippedImage = await page.evaluate(() => {
            const image = document.querySelector(
              ".asset-image-cropper-stage img.cropper-hidden",
            ) as
              | (HTMLImageElement & {
                  cropper?: {
                    getImageData: () => { scaleX: number; scaleY: number };
                  };
                })
              | null;
            if (!image?.cropper) throw new Error("翻转画布尚未准备好");
            const data = image.cropper.getImageData();
            return { scaleX: data.scaleX, scaleY: data.scaleY };
          });
          expect(flippedImage).toEqual({ scaleX: -1, scaleY: -1 });
          const processorControls = page.locator(
            "[data-asset-image-processor-controls]",
          );
          const cropperImage = page.locator(
            ".asset-image-cropper-stage img.cropper-hidden",
          );
          await cropperImage.evaluate((image) => {
            image.setAttribute("data-e2e-cropper-instance", "stable");
          });
          await expect(
            page.getByText("最长边 1920px", { exact: true }),
          ).toHaveCount(1);
          await expect(
            processorControls.getByRole("heading", { name: "裁剪" }),
          ).toBeVisible();
          await expect(
            processorControls.getByRole("heading", { name: "导出" }),
          ).toBeVisible();
          await processorControls
            .getByRole("combobox", { name: "尺寸" })
            .click();
          await page.getByRole("option", { name: "最长边 960px" }).click();
          await expect(cropperImage).toHaveAttribute(
            "data-e2e-cropper-instance",
            "stable",
          );
          await expect(
            page.getByText("最长边 1920px", { exact: true }),
          ).toHaveCount(0);
          await expect(
            page.getByText("最长边 960px", { exact: true }),
          ).toHaveCount(1);
          await expect(
            processorControls.getByText(/^\d+ × \d+$/),
          ).toBeVisible();
          const formatHelp = page.getByRole("button", { name: "输出格式说明" });
          await expect(formatHelp).toBeVisible();
          await expect(
            page.getByText(/WebP 通常体积更小.*资源中心限制校验/),
          ).toHaveCount(0);
          await expect(
            processorControls.getByRole("button", { name: "文件信息" }),
          ).toHaveCount(0);
          await expect(
            processorControls.getByText(inlineName, { exact: true }),
          ).toBeVisible();
          await expect(
            processorControls.getByText(/^160 × 90 ·/),
          ).toBeVisible();
          const inlineFlow = Promise.all([
            page.waitForResponse(
              (response) =>
                response.request().method() === "PUT" &&
                response.url().includes("/asset-api/api/v1/assets/blob/"),
            ),
            page.waitForResponse(
              (response) =>
                response.request().method() === "POST" &&
                response.url().includes("/api/v1/images/finalize"),
            ),
          ]);
          await page.getByRole("button", { name: "使用处理后的图片" }).click();
          const [inlineUpload, inlineFinalize] = await inlineFlow;
          expect(inlineUpload.status()).toBe(200);
          expect(inlineUpload.url()).toContain(
            "/asset-api/api/v1/assets/blob/",
          );
          expect(inlineFinalize.status()).toBe(200);
          await expect(
            page.locator(`.blog-editor-rich-text img[alt="${inlineName}"]`),
          ).toBeVisible();

          const assets = await context.request.get(
            new URL(
              "/asset-api/api/v1/assets?siteKey=blog-main&profileKey=blog-post&page=1&size=100",
              site.url,
            ).toString(),
          );
          expect(assets.ok()).toBeTruthy();
          const assetItems = (await assets.json()).items || [];
          inlineAssetId =
            assetItems.find(
              (asset: { filename?: string }) =>
                asset.filename === inlineProcessedName,
            )?.id || "";
          expect(inlineAssetId).not.toBe("");
          expect(errors).toEqual([]);
        } finally {
          if (coverAssetId && postId) {
            const referenceURL = new URL(
              "/asset-api/api/v1/asset-references",
              site.url,
            );
            referenceURL.searchParams.set("assetId", coverAssetId);
            referenceURL.searchParams.set("siteKey", "blog-main");
            referenceURL.searchParams.set("refType", "post-cover");
            referenceURL.searchParams.set("refId", postId);
            await context.request.delete(referenceURL.toString());
          }
          if (postId) {
            await purgeTestPost(context, site.url, postId);
          }
          if (coverAssetId) {
            await context.request.delete(
              new URL(
                `/asset-api/api/v1/assets/${coverAssetId}`,
                site.url,
              ).toString(),
            );
          }
          if (inlineAssetId) {
            await context.request.delete(
              new URL(
                `/asset-api/api/v1/assets/${inlineAssetId}`,
                site.url,
              ).toString(),
            );
          }
          await context.close();
        }
      });

      test("站点封面比例驱动处理器并允许临时自定义", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let originalConfig: Record<string, unknown> | null = null;
        let postId = "";
        try {
          await page.goto(new URL("/manage", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const home = await context.request.get(
            new URL("/api/v1/home", site.url).toString(),
          );
          expect(home.ok()).toBeTruthy();
          originalConfig = (await home.json()).config;

          await page.goto(
            new URL("/manage/settings?section=site", site.url).toString(),
            { waitUntil: "networkidle" },
          );
          const settingsRatio = page.getByRole("combobox").first();
          await settingsRatio.click();
          await page.getByRole("option", { name: "自定义" }).click();
          await page
            .getByRole("spinbutton", { name: "封面比例宽度" })
            .fill("5");
          await page
            .getByRole("spinbutton", { name: "封面比例高度" })
            .fill("4");
          const settingsUpdate = page.waitForResponse(
            (response) =>
              response.request().method() === "PATCH" &&
              response.url().endsWith("/api/v1/home"),
          );
          await page
            .locator('[data-manage-dock="save"]')
            .getByRole("button", { name: "保存", exact: true })
            .click();
          expect((await settingsUpdate).ok()).toBeTruthy();
          await expect(
            page.getByText("更改已保存", { exact: true }),
          ).toBeVisible();

          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `封面比例验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await page.getByRole("button", { name: "文章设置" }).click();
          await page
            .locator(
              '.blog-editor-settings input[type="file"][accept="image/*"]',
            )
            .setInputFiles({
              name: "ratio-probe.png",
              mimeType: "image/png",
              buffer: Buffer.from(imageFixtureBase64, "base64"),
            });
          const controls = page.locator(
            "[data-asset-image-processor-controls]",
          );
          await expect(controls).toBeVisible();
          const processorRatio = controls.getByRole("combobox").first();
          await expect(processorRatio).toContainText("站点默认 · 5:4");
          await processorRatio.click();
          await expect(
            page.getByRole("option", { name: "横向 3:2" }),
          ).toBeVisible();
          await page.getByRole("option", { name: "自定义比例" }).click();
          await expect(
            page.locator("[data-asset-image-custom-ratio]"),
          ).toBeVisible();
          await page.getByRole("spinbutton", { name: "宽度" }).fill("7");
          await page.getByRole("spinbutton", { name: "高度" }).fill("5");
          await expect(
            page.locator("[data-asset-image-cropper]"),
          ).toHaveAttribute("data-can-confirm", "true");
          await page.getByRole("button", { name: "取消" }).last().click();
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          if (originalConfig) {
            await context.request.patch(
              new URL("/api/v1/home", site.url).toString(),
              { data: originalConfig },
            );
          }
          await context.close();
        }
      });

      test("PNG 超限时可转换、取消并处理二次超限", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        let coverAssetId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `PNG 超限验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await page.getByRole("button", { name: "文章设置" }).click();

          let rejectionsRemaining = 0;
          let rejectionSequence = 0;
          await page.route(`**/api/v1/posts/${postId}/cover`, async (route) => {
            if (
              route.request().method() !== "POST" ||
              rejectionsRemaining < 1
            ) {
              await route.continue();
              return;
            }
            rejectionsRemaining -= 1;
            rejectionSequence += 1;
            const traceId = `asset-limit-${rejectionSequence}`;
            await route.fulfill({
              status: 413,
              headers: {
                "content-type": "application/problem+json",
                "x-trace-id": traceId,
              },
              body: JSON.stringify({
                type: "https://errors.yueli.dev/asset/upload-too-large",
                status: 413,
                code: "asset.upload.too_large",
                params: { maxBytes: 5 * 1024 * 1024 },
                violations: [],
                traceId,
              }),
            });
          });

          const source = Buffer.from(imageFixtureBase64, "base64");
          const png = Buffer.concat([
            source,
            Buffer.alloc(6 * 1024 * 1024 - source.length),
          ]);
          const input = page.locator(
            '.blog-editor-settings input[type="file"][accept="image/*"]',
          );
          const selectPng = () =>
            input.setInputFiles({
              name: "oversized.png",
              mimeType: "image/png",
              buffer: png,
            });
          const processAsPng = async () => {
            const controls = page.locator(
              "[data-asset-image-processor-controls]",
            );
            await expect(controls).toBeVisible();
            await controls.getByRole("combobox", { name: "格式" }).click();
            await page.getByRole("option", { name: /PNG/ }).click();
            await page
              .getByRole("button", { name: "使用处理后的图片" })
              .click();
          };

          rejectionsRemaining = 1;
          await selectPng();
          await processAsPng();
          await expect(
            page.locator("[data-blog-image-compression-dialog]"),
          ).toBeVisible();
          await expect(
            page.getByText("原始大小", { exact: true }),
          ).toBeVisible();
          await expect(page.getByText("5.0 MB", { exact: true })).toBeVisible();
          await page.getByRole("button", { name: "自行处理" }).click();
          await expect(
            page.getByText(
              "已取消转换。请自行压缩后重试，或在资源中心调整封面图片限制。",
              { exact: true },
            ),
          ).toBeVisible();

          rejectionsRemaining = 2;
          await selectPng();
          await processAsPng();
          await page.getByRole("button", { name: "转为 JPEG 并重试" }).click();
          await expect(
            page.getByText(
              "转换后的 JPEG仍超过封面图片上限（5.0 MB）。请自行压缩后重试，或在资源中心调整对应用途的大小限制。",
              { exact: true },
            ),
          ).toBeVisible();

          rejectionsRemaining = 1;
          await selectPng();
          await processAsPng();
          const successfulUpload = Promise.all([
            page.waitForResponse(
              (response) =>
                response.request().method() === "PUT" &&
                response.url().includes("/asset-api/api/v1/assets/blob/"),
            ),
            page.waitForResponse(
              (response) =>
                response.request().method() === "POST" &&
                response
                  .url()
                  .includes(`/api/v1/posts/${postId}/cover/finalize`),
            ),
          ]);
          await page.getByRole("button", { name: "转为 JPEG 并重试" }).click();
          const [upload, finalize] = await successfulUpload;
          expect(upload.status()).toBe(200);
          expect(finalize.status()).toBe(200);
          await expect(page.locator('img[alt="文章封面"]')).toBeVisible();

          const detail = await context.request.get(
            new URL(`/api/v1/posts/${created.post.slug}`, site.url).toString(),
          );
          expect(detail.ok()).toBeTruthy();
          coverAssetId = (await detail.json()).post.coverAssetId || "";
          expect(coverAssetId).not.toBe("");
        } finally {
          if (coverAssetId && postId) {
            const referenceURL = new URL(
              "/asset-api/api/v1/asset-references",
              site.url,
            );
            referenceURL.searchParams.set("assetId", coverAssetId);
            referenceURL.searchParams.set("siteKey", "blog-main");
            referenceURL.searchParams.set("refType", "post-cover");
            referenceURL.searchParams.set("refId", postId);
            await context.request.delete(referenceURL.toString());
          }
          if (postId) {
            await purgeTestPost(context, site.url, postId);
          }
          if (coverAssetId) {
            await context.request.delete(
              new URL(
                `/asset-api/api/v1/assets/${coverAssetId}`,
                site.url,
              ).toString(),
            );
          }
          await context.close();
        }
      });

      test("图片处理器在资源上限前压缩大 PNG", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        let coverAssetId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `真实图片上限验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await page.getByRole("button", { name: "文章设置" }).click();

          const source = Buffer.from(imageFixtureBase64, "base64");
          await page
            .locator(
              '.blog-editor-settings input[type="file"][accept="image/*"]',
            )
            .setInputFiles({
              name: "real-limit.png",
              mimeType: "image/png",
              buffer: Buffer.concat([
                source,
                Buffer.alloc(11 * 1024 * 1024 - source.length),
              ]),
            });
          await expect(
            page.locator("[data-asset-image-processor-controls]"),
          ).toBeVisible();
          await expect(page.getByText(/11 MB/)).toBeVisible();

          const uploadFlow = Promise.all([
            page.waitForResponse(
              (response) =>
                response.request().method() === "PUT" &&
                response.url().includes("/asset-api/api/v1/assets/blob/"),
            ),
            page.waitForResponse(
              (response) =>
                response.request().method() === "POST" &&
                response
                  .url()
                  .includes(`/api/v1/posts/${postId}/cover/finalize`),
            ),
          ]);
          await page.getByRole("button", { name: "使用处理后的图片" }).click();
          const [upload, finalize] = await uploadFlow;
          expect(upload.status()).toBe(200);
          expect(finalize.status()).toBe(200);
          await expect(page.locator('img[alt="文章封面"]')).toBeVisible();
          await expect(
            page.locator("[data-blog-image-compression-dialog]"),
          ).toHaveCount(0);

          const detail = await context.request.get(
            new URL(`/api/v1/posts/${created.post.slug}`, site.url).toString(),
          );
          expect(detail.ok()).toBeTruthy();
          coverAssetId = (await detail.json()).post.coverAssetId || "";
          expect(coverAssetId).not.toBe("");
        } finally {
          if (coverAssetId && postId) {
            const referenceURL = new URL(
              "/asset-api/api/v1/asset-references",
              site.url,
            );
            referenceURL.searchParams.set("assetId", coverAssetId);
            referenceURL.searchParams.set("siteKey", "blog-main");
            referenceURL.searchParams.set("refType", "post-cover");
            referenceURL.searchParams.set("refId", postId);
            await context.request.delete(referenceURL.toString());
          }
          if (postId) {
            await purgeTestPost(context, site.url, postId);
          }
          if (coverAssetId) {
            await context.request.delete(
              new URL(
                `/asset-api/api/v1/assets/${coverAssetId}`,
                site.url,
              ).toString(),
            );
          }
          await context.close();
        }
      });

      test("设置页脏数据保护阻止意外离开", async ({ browser }) => {
        const settings = contract.manage.settings;
        test.skip(!settings, "该产品没有可编辑的通用设置页");
        if (!settings) return;
        const context = await loginE2E(browser);
        const page = await context.newPage();
        const errors = captureErrors(page);
        try {
          await page.goto(new URL(settings.path, site.url).toString(), {
            waitUntil: "domcontentloaded",
          });
          await settleNuxt(page);
          const field = page
            .getByRole("textbox", { name: settings.fieldLabel, exact: false })
            .first();
          await expect(field).toBeVisible();
          await expect(page.locator('[data-manage-dock="save"]')).toHaveCount(
            0,
          );
          await field.fill(`${await field.inputValue()} · 未保存`);
          await expect(page.locator('[data-manage-dock="save"]')).toBeVisible();

          const dialogHandled = new Promise<void>((resolve) => {
            page.once("dialog", async (dialog) => {
              expect(dialog.type()).toBe("confirm");
              expect(dialog.message()).toContain("未保存");
              await dialog.dismiss();
              resolve();
            });
          });
          await page
            .locator(`a[href="${contract.manage.path}"]`)
            .first()
            .click();
          await dialogHandled;
          await expect(page).toHaveURL(
            new RegExp(
              `${settings.path.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}(?:\\?|$)`,
            ),
          );
          expect(errors).toEqual([]);
        } finally {
          await context.close();
        }
      });

      test("空结果状态有明确反馈", async ({ browser, page }) => {
        const context = contract.empty.authenticated
          ? await loginE2E(browser)
          : undefined;
        const targetPage = context ? await context.newPage() : page;
        const errors = captureErrors(targetPage);
        try {
          await targetPage.goto(
            new URL(contract.empty.path, site.url).toString(),
            { waitUntil: "domcontentloaded" },
          );
          const emptyState = targetPage
            .getByText(contract.empty.text, { exact: true })
            .first();
          if (contract.empty.inputPlaceholder) {
            const input = targetPage.getByPlaceholder(
              contract.empty.inputPlaceholder,
            );
            await expect(async () => {
              await input.fill(missing);
              await expect(input).toHaveValue(missing);
              await expect(emptyState).toBeVisible({ timeout: 2_000 });
            }).toPass({ timeout: 15_000, intervals: [250, 500, 1_000] });
          }
          await expect(emptyState).toBeVisible();
          await settleNuxt(targetPage);
          expect(errors).toEqual([]);
        } finally {
          await context?.close();
        }
      });

      test("缺失实体返回产品错误状态", async ({ page }) => {
        const errors = captureErrors(page);
        const response = await page.goto(
          new URL(contract.error.path, site.url).toString(),
          { waitUntil: "domcontentloaded" },
        );
        expect(response?.status()).toBe(contract.error.status);
        await expect(
          page.getByText(contract.error.text, { exact: false }).first(),
        ).toBeVisible();
        await settleNuxt(page);
        expect(
          errors.filter(
            (error) => !error.includes(`status of ${contract.error.status}`),
          ),
        ).toEqual([]);
      });
    });
  }
}
