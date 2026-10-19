package com.talktoode.qa;

import android.app.Instrumentation;
import android.app.UiAutomation;
import android.os.Bundle;
import android.graphics.Rect;
import android.view.accessibility.AccessibilityNodeInfo;

public final class Hierarchy extends Instrumentation {
    private static final String APP = "com.talktoode.app";

    @Override public void onCreate(Bundle arguments) { super.onCreate(arguments); start(); }

    private static String escape(CharSequence value) {
        if (value == null) return "";
        return value.toString().replace("&", "&amp;").replace("\"", "&quot;")
            .replace("<", "&lt;").replace(">", "&gt;").replace("\n", "&#10;").replace("\r", "&#13;");
    }

    private static void append(AccessibilityNodeInfo node, StringBuilder xml, int depth) {
        if (node == null || depth > 100) return;
        boolean owned = APP.contentEquals(node.getPackageName() == null ? "" : node.getPackageName());
        if (owned) {
            Rect bounds = new Rect(); node.getBoundsInScreen(bounds);
            xml.append("<node package=\"").append(APP).append("\" resource-id=\"")
                .append(escape(node.getViewIdResourceName())).append("\" text=\"")
                .append(node.isPassword() ? "[REDACTED]" : escape(node.getText()))
                .append("\" content-desc=\"").append(node.isPassword() ? "[REDACTED]" : escape(node.getContentDescription()))
                .append("\" bounds=\"[").append(bounds.left).append(',').append(bounds.top)
                .append("][").append(bounds.right).append(',').append(bounds.bottom)
                .append("]\" enabled=\"").append(node.isEnabled()).append("\" scrollable=\"")
                .append(node.isScrollable()).append("\" password=\"").append(node.isPassword())
                .append("\" selected=\"").append(node.isSelected())
                .append("\" visible-to-user=\"").append(node.isVisibleToUser()).append("\">");
        }
        for (int i = 0; i < node.getChildCount(); i++) {
            AccessibilityNodeInfo child = node.getChild(i);
            try { append(child, xml, depth + 1); } finally { if (child != null) child.recycle(); }
        }
        if (owned) xml.append("</node>");
    }

    @Override public void onStart() {
        Bundle result = new Bundle();
        try {
            UiAutomation automation = getUiAutomation(UiAutomation.FLAG_DONT_SUPPRESS_ACCESSIBILITY_SERVICES);
            AccessibilityNodeInfo node = null;
            for (int attempt = 0; attempt < 20 && node == null; attempt++) {
                node = automation.getRootInActiveWindow();
                if (node == null) android.os.SystemClock.sleep(100);
            }
            try {
                if (node == null || !APP.contentEquals(node.getPackageName() == null ? "" : node.getPackageName())) {
                    result.putString("ode_error", "Ode does not own the active window"); finish(1, result); return;
                }
                StringBuilder xml = new StringBuilder("<hierarchy>");
                append(node, xml, 0); xml.append("</hierarchy>");
                result.putString("ode_hierarchy", xml.toString());
                finish(0, result);
            } finally { if (node != null) node.recycle(); }
        } catch (Exception error) {
            result.putString("ode_error", error.getClass().getSimpleName()); finish(1, result);
        }
    }
}
