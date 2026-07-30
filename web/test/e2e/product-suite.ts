import { registerAccessibilitySuite } from "./accessibility-matrix.spec";
import { registerJourneySuite } from "./site-matrix.spec";
import { registerPerformanceSuite } from "./performance-matrix.spec";
import { registerVisualSuite } from "./visual-matrix.spec";

export function registerProductSuite(product: string) {
  const suite = process.env.BLOG_E2E_SUITE?.trim() || "all";
  if (suite === "all" || suite === "journeys") registerJourneySuite(product);
  if (suite === "all" || suite === "visual") registerVisualSuite(product);
  if (suite === "all" || suite === "accessibility")
    registerAccessibilitySuite(product);
  if (suite === "all" || suite === "performance")
    registerPerformanceSuite(product);
}
