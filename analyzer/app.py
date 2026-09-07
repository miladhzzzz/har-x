"""
HAR-X Analyzer — interactive HAR exploration in Streamlit.

Reads HAR files produced by the Go capture tool (harx-capture) or any
standard browser export, and gives you the same kind of view as browser
DevTools' Network tab: a waterfall, filters, per-request detail, and
summary stats — but interactive and shareable as a page instead of a
static PNG.

Run:
    streamlit run app.py
"""
from __future__ import annotations

import glob
import json
import os
from datetime import datetime

import pandas as pd
import plotly.express as px
import plotly.graph_objects as go
import streamlit as st

CAPTURES_DIR = os.environ.get("HARX_CAPTURES_DIR", os.path.join(os.path.dirname(__file__), "..", "captures"))

st.set_page_config(page_title="HAR-X Analyzer", layout="wide")


# --------------------------------------------------------------------------
# Loading
# --------------------------------------------------------------------------

@st.cache_data(show_spinner=False)
def load_har(path_or_bytes) -> dict:
    if isinstance(path_or_bytes, (bytes, bytearray)):
        return json.loads(path_or_bytes)
    with open(path_or_bytes, "r", encoding="utf-8") as f:
        return json.load(f)


def domain_of(url: str) -> str:
    try:
        return url.split("://", 1)[1].split("/", 1)[0]
    except IndexError:
        return url


def har_to_dataframe(har: dict) -> pd.DataFrame:
    entries = har.get("log", {}).get("entries", [])
    pages = {p["id"]: p.get("title", p["id"]) for p in har.get("log", {}).get("pages", [])}

    rows = []
    for i, e in enumerate(entries):
        req = e.get("request", {})
        resp = e.get("response", {})
        content = resp.get("content", {})
        timings = e.get("timings", {})
        url = req.get("url", "")
        rows.append({
            "#": i + 1,
            "page": pages.get(e.get("pageref"), e.get("pageref", "")),
            "started": e.get("startedDateTime", ""),
            "method": req.get("method", ""),
            "url": url,
            "domain": domain_of(url),
            "status": resp.get("status", 0),
            "resource_type": e.get("_resourceType", "") or "Other",
            "mime_type": content.get("mimeType", ""),
            "time_ms": e.get("time", 0.0),
            "size_bytes": content.get("size", 0) or 0,
            "blocked_ms": timings.get("blocked", -1),
            "dns_ms": timings.get("dns", -1),
            "connect_ms": timings.get("connect", -1),
            "ssl_ms": timings.get("ssl", -1),
            "send_ms": timings.get("send", -1),
            "wait_ms": timings.get("wait", -1),
            "receive_ms": timings.get("receive", -1),
            "error": e.get("_error", ""),
        })
    df = pd.DataFrame(rows)
    if not df.empty:
        df["started_dt"] = pd.to_datetime(df["started"], errors="coerce")
        df = df.sort_values("started_dt").reset_index(drop=True)
        df["#"] = range(1, len(df) + 1)
    return df


def discover_capture_files() -> list[str]:
    pattern = os.path.join(CAPTURES_DIR, "*.har")
    return sorted(glob.glob(pattern), key=os.path.getmtime, reverse=True)


# --------------------------------------------------------------------------
# Sidebar: file selection + filters
# --------------------------------------------------------------------------

st.sidebar.title("HAR-X Analyzer")

files = discover_capture_files()
har_bytes = None
chosen_label = None

source = st.sidebar.radio("Source", ["Captured files", "Upload a .har"], horizontal=False)

if source == "Captured files":
    if not files:
        st.sidebar.info(f"No .har files found in {os.path.abspath(CAPTURES_DIR)}. "
                         f"Run harx-capture, or switch to 'Upload a .har'.")
    else:
        labels = [f"{os.path.basename(f)}  ({datetime.fromtimestamp(os.path.getmtime(f)):%Y-%m-%d %H:%M})" for f in files]
        idx = st.sidebar.selectbox("Capture", range(len(files)), format_func=lambda i: labels[i])
        chosen_label = files[idx]
else:
    uploaded = st.sidebar.file_uploader("Upload .har", type=["har", "json"])
    if uploaded is not None:
        har_bytes = uploaded.read()
        chosen_label = uploaded.name

if har_bytes is None and chosen_label is None:
    st.title("HAR-X Analyzer")
    st.write("Select or upload a HAR capture from the sidebar to get started.")
    st.stop()

har = load_har(har_bytes if har_bytes is not None else chosen_label)
df = har_to_dataframe(har)

if df.empty:
    st.warning("This HAR file has no entries.")
    st.stop()

st.sidebar.markdown("---")
st.sidebar.subheader("Filters")

status_min, status_max = int(df["status"].min()), int(df["status"].max())
status_range = st.sidebar.slider("Status code range", 0, max(599, status_max), (0, max(599, status_max)))

methods = sorted(df["method"].unique())
sel_methods = st.sidebar.multiselect("Method", methods, default=methods)

rtypes = sorted(df["resource_type"].unique())
sel_rtypes = st.sidebar.multiselect("Resource type", rtypes, default=rtypes)

domains = sorted(df["domain"].unique())
sel_domains = st.sidebar.multiselect("Domain", domains, default=domains)

mime_filter = st.sidebar.text_input("Content-type contains", "")

fdf = df[
    df["status"].between(status_range[0], status_range[1])
    & df["method"].isin(sel_methods)
    & df["resource_type"].isin(sel_rtypes)
    & df["domain"].isin(sel_domains)
]
if mime_filter:
    fdf = fdf[fdf["mime_type"].str.contains(mime_filter, case=False, na=False)]

# --------------------------------------------------------------------------
# Header + summary metrics
# --------------------------------------------------------------------------

st.title("HAR-X Analyzer")
st.caption(os.path.basename(chosen_label) if chosen_label else "uploaded capture")

failed = int((fdf["status"] >= 400).sum() | (fdf["status"] == 0).sum())
c1, c2, c3, c4, c5 = st.columns(5)
c1.metric("Requests", len(fdf))
c2.metric("Failed / error", failed)
c3.metric("Avg time", f"{fdf['time_ms'].mean():.0f} ms" if len(fdf) else "-")
c4.metric("Slowest", f"{fdf['time_ms'].max():.0f} ms" if len(fdf) else "-")
c5.metric("Total transferred", f"{fdf['size_bytes'].sum() / 1024:.1f} KB")

st.markdown("---")

# --------------------------------------------------------------------------
# Waterfall
# --------------------------------------------------------------------------

st.subheader("Waterfall")

wdf = fdf.copy()
if not wdf.empty:
    t0 = wdf["started_dt"].min()
    wdf["offset_ms"] = (wdf["started_dt"] - t0).dt.total_seconds() * 1000
    wdf["label"] = wdf["#"].astype(str) + ". " + wdf["url"].str.slice(0, 70)

    phases = ["blocked_ms", "dns_ms", "connect_ms", "ssl_ms", "send_ms", "wait_ms", "receive_ms"]
    colors = {
        "blocked_ms": "#c7c7c7", "dns_ms": "#8ecae6", "connect_ms": "#219ebc",
        "ssl_ms": "#ffb703", "send_ms": "#fb8500", "wait_ms": "#a663cc", "receive_ms": "#43aa8b",
    }

    fig = go.Figure()
    running_start = wdf["offset_ms"].copy()
    for phase in phases:
        vals = wdf[phase].clip(lower=0)
        fig.add_trace(go.Bar(
            y=wdf["label"], x=vals, base=running_start, orientation="h",
            name=phase.replace("_ms", ""), marker_color=colors[phase],
            hovertemplate=f"{phase.replace('_ms','')}: %{{x:.1f}} ms<extra></extra>",
        ))
        running_start = running_start + vals

    row_h = 22
    fig.update_layout(
        barmode="stack", height=max(300, min(1400, row_h * len(wdf) + 100)),
        yaxis=dict(autorange="reversed"), xaxis_title="ms since first request",
        legend_title="phase", margin=dict(l=10, r=10, t=10, b=10),
    )
    st.plotly_chart(fig, use_container_width=True)
else:
    st.write("No requests match the current filters.")

# --------------------------------------------------------------------------
# Distributions
# --------------------------------------------------------------------------

col1, col2 = st.columns(2)
with col1:
    st.subheader("Status codes")
    sc = fdf["status"].astype(str).value_counts().reset_index()
    sc.columns = ["status", "count"]
    st.plotly_chart(px.bar(sc, x="status", y="count"), use_container_width=True)

with col2:
    st.subheader("Content types")
    ct = fdf["mime_type"].replace("", "(none)").value_counts().reset_index()
    ct.columns = ["mime_type", "count"]
    st.plotly_chart(px.pie(ct, names="mime_type", values="count"), use_container_width=True)

col3, col4 = st.columns(2)
with col3:
    st.subheader("Time vs size")
    st.plotly_chart(
        px.scatter(fdf, x="time_ms", y="size_bytes", color="resource_type",
                   hover_data=["url", "status"]),
        use_container_width=True,
    )
with col4:
    st.subheader("Time by resource type")
    st.plotly_chart(px.box(fdf, x="resource_type", y="time_ms"), use_container_width=True)

st.markdown("---")

# --------------------------------------------------------------------------
# Table + detail
# --------------------------------------------------------------------------

st.subheader("Requests")
table_cols = ["#", "method", "status", "resource_type", "domain", "url", "time_ms", "size_bytes", "mime_type"]
st.dataframe(fdf[table_cols], use_container_width=True, height=350)

st.subheader("Request detail")
if not fdf.empty:
    sel = st.selectbox("Pick a request by #", fdf["#"].tolist())
    entry = har["log"]["entries"][int(sel) - 1]
    req, resp = entry.get("request", {}), entry.get("response", {})

    d1, d2 = st.columns(2)
    with d1:
        st.markdown(f"**{req.get('method')} {resp.get('status')}**  \n{req.get('url')}")
        st.markdown("**Request headers**")
        st.json({h["name"]: h["value"] for h in req.get("headers", [])})
        if req.get("postData"):
            st.markdown("**Request body**")
            st.code(req["postData"].get("text", ""))
    with d2:
        st.markdown("**Response headers**")
        st.json({h["name"]: h["value"] for h in resp.get("headers", [])})
        body = resp.get("content", {}).get("text", "")
        if body:
            st.markdown("**Response body (preview)**")
            st.code(body[:5000])

    st.markdown("**Timing breakdown (ms)**")
    st.json(entry.get("timings", {}))
