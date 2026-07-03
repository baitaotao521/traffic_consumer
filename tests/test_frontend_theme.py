#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""验证 Vue 控制台只使用项目指定的主题 token。"""

import re
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
FRONTEND_SRC = REPO_ROOT / "frontend" / "src"
FRONTEND_DASHBOARD = FRONTEND_SRC / "components" / "dashboard"
FRONTEND_STYLE = FRONTEND_SRC / "style.css"
COMPILED_CSS = REPO_ROOT / "static" / "app" / "assets" / "app.css"
COMPILED_JS = REPO_ROOT / "static" / "app" / "assets" / "app.js"

REQUESTED_ROOT_THEME = """:root {
  --background: oklch(1 0 0);
  --foreground: oklch(0.145 0 0);
  --card: oklch(1 0 0);
  --card-foreground: oklch(0.145 0 0);
  --popover: oklch(1 0 0);
  --popover-foreground: oklch(0.145 0 0);
  --primary: oklch(0.205 0 0);
  --primary-foreground: oklch(0.985 0 0);
  --secondary: oklch(0.97 0 0);
  --secondary-foreground: oklch(0.205 0 0);
  --muted: oklch(0.97 0 0);
  --muted-foreground: oklch(0.556 0 0);
  --accent: oklch(0.97 0 0);
  --accent-foreground: oklch(0.205 0 0);
  --destructive: oklch(0.577 0.245 27.325);
  --border: oklch(0.922 0 0);
  --input: oklch(0.922 0 0);
  --ring: oklch(0.708 0 0);
  --chart-1: oklch(0.87 0 0);
  --chart-2: oklch(0.556 0 0);
  --chart-3: oklch(0.439 0 0);
  --chart-4: oklch(0.371 0 0);
  --chart-5: oklch(0.269 0 0);
  --radius: 0.625rem;
  --sidebar: oklch(0.985 0 0);
  --sidebar-foreground: oklch(0.145 0 0);
  --sidebar-primary: oklch(0.205 0 0);
  --sidebar-primary-foreground: oklch(0.985 0 0);
  --sidebar-accent: oklch(0.97 0 0);
  --sidebar-accent-foreground: oklch(0.205 0 0);
  --sidebar-border: oklch(0.922 0 0);
  --sidebar-ring: oklch(0.708 0 0);
}"""

REQUESTED_DARK_THEME = """.dark {
  --background: oklch(0.145 0 0);
  --foreground: oklch(0.985 0 0);
  --card: oklch(0.205 0 0);
  --card-foreground: oklch(0.985 0 0);
  --popover: oklch(0.205 0 0);
  --popover-foreground: oklch(0.985 0 0);
  --primary: oklch(0.922 0 0);
  --primary-foreground: oklch(0.205 0 0);
  --secondary: oklch(0.269 0 0);
  --secondary-foreground: oklch(0.985 0 0);
  --muted: oklch(0.269 0 0);
  --muted-foreground: oklch(0.708 0 0);
  --accent: oklch(0.269 0 0);
  --accent-foreground: oklch(0.985 0 0);
  --destructive: oklch(0.704 0.191 22.216);
  --border: oklch(1 0 0 / 10%);
  --input: oklch(1 0 0 / 15%);
  --ring: oklch(0.556 0 0);
  --chart-1: oklch(0.87 0 0);
  --chart-2: oklch(0.556 0 0);
  --chart-3: oklch(0.439 0 0);
  --chart-4: oklch(0.371 0 0);
  --chart-5: oklch(0.269 0 0);
  --sidebar: oklch(0.205 0 0);
  --sidebar-foreground: oklch(0.985 0 0);
  --sidebar-primary: oklch(0.488 0.243 264.376);
  --sidebar-primary-foreground: oklch(0.985 0 0);
  --sidebar-accent: oklch(0.269 0 0);
  --sidebar-accent-foreground: oklch(0.985 0 0);
  --sidebar-border: oklch(1 0 0 / 10%);
  --sidebar-ring: oklch(0.556 0 0);
}"""

FORBIDDEN_SOURCE_PATTERNS = {
    "硬编码十六进制颜色": re.compile(r"#[0-9a-fA-F]{3,8}\b"),
    "rgba 颜色": re.compile(r"rgba\("),
    "任意颜色工具类": re.compile(r"\b(?:bg|text|border)-\[#"),
    "旧 dashboard 自定义样式": re.compile(r"\bdashboard-"),
    "Tailwind 默认色系工具类": re.compile(
        r"\b(?:bg|text|border|from|to|via|ring|outline|fill|stroke)-"
        r"(?:slate|rose|amber|emerald|blue)(?:-\d|/|\b)"
    ),
    "Tailwind 黑白色工具类": re.compile(
        r"\b(?:bg|text|border|from|to|via|ring|outline|fill|stroke)-(?:white|black)(?:/|\b)"
    ),
}

FORBIDDEN_BUNDLE_FRAGMENTS = (
    "--color-slate-",
    "--color-rose-",
    "--color-amber-",
    "--color-emerald-",
    "--color-blue-",
    "#2563eb",
    "#d8dee9",
    "#eef2f7",
    "dashboard-reveal",
    "dashboard-surface",
)

REDUNDANT_DASHBOARD_CLASS_FRAGMENTS = (
    "border-border",
    "border-none",
    "bg-card",
    "bg-popover",
    "shadow-none",
    "text-foreground",
    "data-[state=unchecked]:bg-input",
)


def _read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def _css_block(css: str, selector: str) -> str:
    match = re.search(rf"(?ms)^{re.escape(selector)} \{{\n.*?^}}", css)
    if not match:
        raise AssertionError(f"未找到 {selector} 主题块")
    return match.group(0)


class FrontendThemeTests(unittest.TestCase):
    """锁定前端主题 token，避免默认色系和遗留样式重新混入。"""

    def test_style_uses_requested_theme_blocks(self):
        """源码主题块必须与指定的亮色和暗色 token 完全一致。"""
        css = _read(FRONTEND_STYLE)
        self.assertEqual(_css_block(css, ":root"), REQUESTED_ROOT_THEME)
        self.assertEqual(_css_block(css, ".dark"), REQUESTED_DARK_THEME)

    def test_tailwind_theme_clears_default_colors(self):
        """Tailwind 默认颜色命名空间必须先清空，只保留项目 token 映射。"""
        css = _read(FRONTEND_STYLE)
        theme = _css_block(css, "@theme inline")
        self.assertIn("--color-*: initial;", theme)
        for fragment in ("--color-slate", "--color-rose", "--color-amber", "--color-emerald", "--color-blue"):
            self.assertNotIn(fragment, theme)

    def test_frontend_source_uses_theme_colors_only(self):
        """Vue 源码只能通过主题 token 取色，不能绕回旧色值或默认色系。"""
        checked_suffixes = {".css", ".ts", ".vue"}
        for path in FRONTEND_SRC.rglob("*"):
            if path.suffix not in checked_suffixes:
                continue
            text = _read(path)
            for label, pattern in FORBIDDEN_SOURCE_PATTERNS.items():
                match = pattern.search(text)
                if match:
                    relative = path.relative_to(REPO_ROOT)
                    self.fail(f"{relative} 包含{label}: {match.group(0)}")

    def test_dashboard_does_not_redeclare_component_default_styles(self):
        """Dashboard 调用处不重复手写组件默认主题 class，只保留布局和业务状态样式。"""
        checked_paths = list(FRONTEND_DASHBOARD.rglob("*.vue"))
        checked_paths.append(FRONTEND_SRC / "composables" / "useDashboard.ts")
        for path in checked_paths:
            text = _read(path)
            for fragment in REDUNDANT_DASHBOARD_CLASS_FRAGMENTS:
                if fragment in text:
                    relative = path.relative_to(REPO_ROOT)
                    self.fail(f"{relative} 重复声明组件默认样式: {fragment}")

    def test_compiled_bundle_does_not_reintroduce_removed_theme_values(self):
        """编译产物也不能重新出现已移除的默认色系和旧硬编码色值。"""
        bundle = _read(COMPILED_CSS) + "\n" + _read(COMPILED_JS)
        for fragment in FORBIDDEN_BUNDLE_FRAGMENTS:
            self.assertNotIn(fragment, bundle)


if __name__ == "__main__":
    unittest.main()
