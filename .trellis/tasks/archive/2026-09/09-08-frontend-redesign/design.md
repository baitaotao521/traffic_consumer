# Technical design

- App.vue owns hash navigation and the existing useDashboard instance; section switches do not restart polling or lose editor state.
- Official shadcn Sidebar provides desktop navigation and mobile Sheet behavior. Product components compose primitives.
- Keep chart/dialog lazy boundaries. SpeedPanel becomes a trend card and two secondary cards.
- PlanPanel owns only local search/filter presentation; emits existing typed business actions.
- ConfigSheet uses tabs around existing validated fields; errors select the owning tab.
- Light semantic theme in style.css is the single visual source; do not restyle primitive internals.
- Backend/SDK/domain serialization are unchanged. Rollback is frontend source plus a fresh build.
