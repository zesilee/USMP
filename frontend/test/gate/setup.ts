// gate 套件独立装配：只挂 jest-dom 断言与 matchMedia polyfill，
// 不 import @testing-library/react（其 CJS 子模块会原生 require 真 react-dom）。
import '@testing-library/jest-dom/vitest'

if (typeof window !== 'undefined' && !window.matchMedia) {
  window.matchMedia = ((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia
}
