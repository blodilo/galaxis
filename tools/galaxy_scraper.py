#!/usr/bin/env python3
"""
galaxy_scraper.py — BL-03 Foto-Template Morphologie
Galaxis Projekt, 2026-03-17

Bevölkert galaxy_morphology_catalog_v1.0.yaml mit verifizierten Bildern
für alle de-Vaucouleurs-Morphologietypen.

Quellen:
  Discovery:  SIMBAD TAP (astroquery)
  Bilder:     SDSS SkyServer Cutout-Dienst (Primär)
              SkyView / DSS2 (Fallback Südhimmel / kleine Galaxien)
  QA:         Gemini Vision API (gemini-2.0-flash)

Lizenzen der Bilddaten:
  SDSS DR17:  CC BY 4.0 (sdss.org/collaboration/citing-sdss)
  DSS2:       Public Domain / STScI

Nutzung:
  export GEMINI_API_KEY=...   (oder tools/.env)
  python tools/galaxy_scraper.py                        # alle Typen
  python tools/galaxy_scraper.py --type Sb SBb E3       # einzelne Typen
  python tools/galaxy_scraper.py --type Sb --dry-run    # kein Download
  python tools/galaxy_scraper.py --reset                # archiviert alte Einträge
"""

import os
import re
import sys
import time
import shutil
import logging
import argparse
from pathlib import Path
from io import BytesIO
from typing import Optional

import yaml
import numpy as np
import requests
from PIL import Image
from dotenv import load_dotenv
import google.generativeai as genai
from astroquery.simbad import Simbad
from astroquery.skyview import SkyView
import astropy.units as u
from astropy.coordinates import SkyCoord
from astropy.visualization import ZScaleInterval

# ── Pfade ─────────────────────────────────────────────────────────────────────
TOOLS_DIR   = Path(__file__).parent
BASE_DIR    = TOOLS_DIR.parent
CATALOG     = BASE_DIR / "galaxy_morphology_catalog_v1.0.yaml"
ASSETS_DIR  = BASE_DIR / "assets" / "morphology"
ARCHIVE_DIR = ASSETS_DIR / "archive"

# ── Konstanten ────────────────────────────────────────────────────────────────
IMAGES_PER_TYPE  = 5
MIN_AXIS_ARCMIN  = 0.8    # Kleinste akzeptierte Scheibengröße (arcmin)
MAX_AXIS_ARCMIN  = 25.0   # Größte (zu große Galaxien schwer zu rahmen)
BA_FACE_ON       = 0.7    # Mindest-Achsenverhältnis b/a für face-on-Filter
IMG_OUT_PX       = 2048   # Ausgabebreite in Pixeln
IMG_MAX_PX       = 4096   # Maximale gespeicherte Breite
SIMBAD_LIMIT     = 300    # Maximale Kandidaten pro Typ aus SIMBAD
REQUEST_DELAY    = 0.6    # Pause zwischen HTTP-Anfragen (Sekunden)

# ── De-Vaucouleurs Typentabelle ───────────────────────────────────────────────
# query:    SIMBAD morph_type LIKE-Pattern (Vorabselektion; Python-Seite verfeinert)
# b_a_min:  Für face-on-Filter (Spiralen) oder Elliptizitäts-Untergrenze (E-Typen)
# b_a_max:  Obere Grenze (nur E-Typen zur E0-E7-Klassifikation per Achsenverhältnis)
# face_on:  True = b/a-Filter anwenden
HUBBLE_TYPES: dict[str, dict] = {
    # ── Elliptisch (E0-E7 über Achsenverhältnis b/a klassifiziert) ─────────────
    "E0": {"query": "E%", "b_a_min": 0.93, "b_a_max": 1.01, "face_on": False},
    "E1": {"query": "E%", "b_a_min": 0.83, "b_a_max": 0.93, "face_on": False},
    "E2": {"query": "E%", "b_a_min": 0.73, "b_a_max": 0.83, "face_on": False},
    "E3": {"query": "E%", "b_a_min": 0.63, "b_a_max": 0.73, "face_on": False},
    "E4": {"query": "E%", "b_a_min": 0.53, "b_a_max": 0.63, "face_on": False},
    "E5": {"query": "E%", "b_a_min": 0.43, "b_a_max": 0.53, "face_on": False},
    "E6": {"query": "E%", "b_a_min": 0.33, "b_a_max": 0.43, "face_on": False},
    "E7": {"query": "E%", "b_a_min": 0.23, "b_a_max": 0.33, "face_on": False},
    # ── Lentikulär ─────────────────────────────────────────────────────────────
    "S0m": {"query": "S0-%",   "face_on": True,  "b_a_min": 0.7},
    "S0":  {"query": "SA0%",   "face_on": True,  "b_a_min": 0.7},
    "S0p": {"query": "S0+%",   "face_on": True,  "b_a_min": 0.7},
    "S0a": {"query": "S0/a%",  "face_on": True,  "b_a_min": 0.7},
    # ── Spiralen (unbarred SA) ─────────────────────────────────────────────────
    "Sa":  {"query": "SA%a%",  "face_on": True,  "b_a_min": 0.7},
    "Sab": {"query": "SA%ab%", "face_on": True,  "b_a_min": 0.7},
    "Sb":  {"query": "SA%b%",  "face_on": True,  "b_a_min": 0.7},
    "Sbc": {"query": "SA%bc%", "face_on": True,  "b_a_min": 0.7},
    "Sc":  {"query": "SA%c%",  "face_on": True,  "b_a_min": 0.7},
    "Scd": {"query": "SA%cd%", "face_on": True,  "b_a_min": 0.7},
    "Sd":  {"query": "SA%d%",  "face_on": True,  "b_a_min": 0.7},
    "Sdm": {"query": "SA%dm%", "face_on": True,  "b_a_min": 0.7},
    "Sm":  {"query": "SA%m%",  "face_on": True,  "b_a_min": 0.7},
    # ── Balken-Spiralen (SB + SAB → beide als SB gezählt) ─────────────────────
    "SBa":  {"query": "SB%a%",  "face_on": True, "b_a_min": 0.7},
    "SBab": {"query": "SB%ab%", "face_on": True, "b_a_min": 0.7},
    "SBb":  {"query": "SB%b%",  "face_on": True, "b_a_min": 0.7},
    "SBbc": {"query": "SB%bc%", "face_on": True, "b_a_min": 0.7},
    "SBc":  {"query": "SB%c%",  "face_on": True, "b_a_min": 0.7},
    "SBcd": {"query": "SB%cd%", "face_on": True, "b_a_min": 0.7},
    "SBd":  {"query": "SB%d%",  "face_on": True, "b_a_min": 0.7},
    "SBdm": {"query": "SB%dm%", "face_on": True, "b_a_min": 0.7},
    "SBm":  {"query": "SB%m%",  "face_on": True, "b_a_min": 0.7},
    # ── Irreguläre ─────────────────────────────────────────────────────────────
    "Im":  {"query": "Im%",  "face_on": False},
    "IBm": {"query": "IB%m", "face_on": False},
    "I0":  {"query": "I0%",  "face_on": False},
}


# ── RC3/SIMBAD morph_type Parser ─────────────────────────────────────────────

def parse_rc3_type(morph: str) -> Optional[dict]:
    """
    Parst einen RC3/SIMBAD morph_type-String.
    Gibt dict mit keys: family, bar_class, subtype zurück oder None.

    Beispiele:
      "SA(s)bc"  → family="S", bar_class="SA", subtype="bc"
      "SB(r)b"   → family="S", bar_class="SB", subtype="b"
      "SAB(rs)b" → family="S", bar_class="SB", subtype="b"  (SAB → SB)
      "E3"       → family="E"
      "SA0"      → family="S0", bar_class="SA"
      "IB(s)m"   → family="Irr", bar_class="IB", subtype="m"
    """
    m = morph.strip()
    if not m or m in ("--", "?", ""):
        return None

    # Elliptisch
    if re.match(r"^E\d?", m, re.IGNORECASE):
        return {"family": "E", "bar_class": None, "subtype": None}

    # Lentikulär: S[A|B|AB]?0[+-]?[/a]?
    s0_m = re.match(r"^S(A|AB|B)?0([-+])?(/a)?", m, re.IGNORECASE)
    if s0_m:
        return {
            "family": "S0",
            "bar_class": (s0_m.group(1) or "SA").upper(),
            "modifier": s0_m.group(2),
            "s0a": bool(s0_m.group(3)),
        }

    # Irregulär: IB...m / Im / I0
    irr_m = re.match(r"^(IB?A?B?)\s*(?:\([^)]*\))?\s*([a-z0-9]*)", m, re.IGNORECASE)
    if irr_m and m[0].upper() == "I":
        bar = "IB" if "B" in irr_m.group(1).upper() else "I"
        subtype = irr_m.group(2).lower() or None
        return {"family": "Irr", "bar_class": bar, "subtype": subtype}

    # Spiralen: (SA|SAB|SB|S) optionale_Klammer(n) subtyp
    spiral_m = re.match(
        r"^(SB|SAB|SA|S)\s*(?:\([^)]+\))?\s*([a-z]{1,2})",
        m, re.IGNORECASE
    )
    if spiral_m:
        bar_raw = spiral_m.group(1).upper()
        subtype = spiral_m.group(2).lower()
        if subtype not in {"a", "ab", "b", "bc", "c", "cd", "d", "dm", "m"}:
            return None
        # SAB (intermediär) → als SB werten (Balken sichtbar)
        bar_class = "SB" if bar_raw in ("SB", "SAB") else "SA"
        return {"family": "S", "bar_class": bar_class, "subtype": subtype}

    return None


def matches_hubble_type(morph: str, target: str) -> bool:
    """Prüft ob ein SIMBAD morph_type-String zum Ziel-de-Vaucouleurs-Typ passt."""
    p = parse_rc3_type(morph)
    if p is None:
        return False

    # ── Elliptisch ─────────────────────────────────────────────────────────────
    if target.startswith("E"):
        return p["family"] == "E"  # E0-E7 über Achsenverhältnis klassifiziert

    # ── Lentikulär ─────────────────────────────────────────────────────────────
    if target == "S0m":
        return p["family"] == "S0" and p.get("modifier") == "-"
    if target == "S0":
        return p["family"] == "S0" and not p.get("modifier") and not p.get("s0a")
    if target == "S0p":
        return p["family"] == "S0" and p.get("modifier") == "+"
    if target == "S0a":
        return p["family"] == "S0" and bool(p.get("s0a"))

    # ── Irreguläre ─────────────────────────────────────────────────────────────
    if target == "Im":
        return p["family"] == "Irr" and p.get("bar_class") == "I" and p.get("subtype") == "m"
    if target == "IBm":
        return p["family"] == "Irr" and p.get("bar_class") == "IB"
    if target == "I0":
        return p["family"] == "Irr" and p.get("subtype") == "0"

    # ── Spiralen ───────────────────────────────────────────────────────────────
    if p["family"] != "S":
        return False
    subtype = p["subtype"]
    bar = p["bar_class"]

    if target.startswith("SB"):
        return bar == "SB" and target[2:].lower() == subtype
    else:  # Sa, Sb, Sc, ... (unbarred)
        return bar == "SA" and target[1:].lower() == subtype


# ── Bild-Download ─────────────────────────────────────────────────────────────

def _is_blank_image(img: Image.Image) -> bool:
    """Gibt True zurück wenn das Bild leer/schwarz/uniform ist."""
    arr = np.array(img.convert("L"), dtype=float)
    bright_fraction = (arr > 25).mean()
    return bright_fraction < 0.03  # weniger als 3% helle Pixel → leer


def download_sdss(ra: float, dec: float, maj_arcmin: float) -> Optional[Image.Image]:
    """
    Lädt SDSS DR17 Farb-Composite-Bild herunter.
    Gibt None zurück wenn SDSS-Abdeckung fehlt oder Bild leer ist.
    """
    # Maßstab: Galaxie füllt ~40% des Frames (2x Durchmesser pro Seite)
    scale = max(0.1, (maj_arcmin * 60 * 2.5) / IMG_OUT_PX)
    url = (
        f"https://skyserver.sdss.org/dr17/SkyServerWS/ImgCutout/getjpeg"
        f"?ra={ra:.6f}&dec={dec:.6f}&scale={scale:.4f}"
        f"&width={IMG_OUT_PX}&height={IMG_OUT_PX}"
    )
    try:
        resp = requests.get(url, timeout=30)
        resp.raise_for_status()
        img = Image.open(BytesIO(resp.content)).convert("RGB")
        if _is_blank_image(img):
            return None
        return img
    except Exception as e:
        logging.debug(f"SDSS failed RA={ra:.3f} DEC={dec:.3f}: {e}")
        return None


def download_skyview(ra: float, dec: float, maj_arcmin: float) -> Optional[Image.Image]:
    """
    Fallback: DSS2 Red via SkyView. Gibt Graustufen-RGB zurück.
    Geeignet für Südhimmel und Galaxien außerhalb des SDSS-Footprints.
    """
    try:
        radius = min(maj_arcmin * 2.5, 30.0) * u.arcmin
        images = SkyView.get_images(
            position=SkyCoord(ra=ra * u.deg, dec=dec * u.deg),
            survey=["DSS2 Red"],
            radius=radius,
            pixels=IMG_OUT_PX,
        )
        if not images:
            return None
        fits_data = images[0][0].data.astype(float)
        interval = ZScaleInterval()
        vmin, vmax = interval.get_limits(fits_data)
        if vmax <= vmin:
            return None
        normalized = np.clip((fits_data - vmin) / (vmax - vmin), 0, 1)
        rgb = (normalized * 255).astype(np.uint8)
        img = Image.fromarray(rgb).convert("RGB")
        if _is_blank_image(img):
            return None
        return img
    except Exception as e:
        logging.debug(f"SkyView failed RA={ra:.3f} DEC={dec:.3f}: {e}")
        return None


# ── Gemini Vision QA ──────────────────────────────────────────────────────────

def gemini_quality_check(img: Image.Image, hubble_type: str) -> tuple[bool, str]:
    """
    Bewertet ein Galaxienbild mit Gemini Vision.
    Gibt (accept: bool, reason: str) zurück.
    """
    prompt = f"""Evaluate this astronomical image as a density-map template for a galaxy morphology generator.
Target de Vaucouleurs type: {hubble_type}

Answer with exactly one of: ACCEPT, REJECT, or UNCERTAIN
Followed by a colon and one sentence explaining why.

ACCEPT criteria (all must hold):
- Galaxy clearly visible and roughly centered
- Face-on or near-face-on orientation (not edge-on, not strongly tilted) — skip this for E-types and Irr
- Morphological features distinguishable (arms/bar/smooth ellipse/irregular clumps)
- Not dominated by foreground stars, dust lanes, or image artifacts
- Morphology visually consistent with type {hubble_type}

REJECT criteria (any one suffices):
- Galaxy is edge-on or strongly inclined (inclination > ~45°) — spirals/lenticulars only
- Galaxy too small (<5% of frame) or too faint to see structure
- Clear morphology mismatch with {hubble_type}
- Heavy artifacts: saturation, missing stripes, CCD bleeding

Format: ACCEPT: <reason>  or  REJECT: <reason>  or  UNCERTAIN: <reason>"""

    model = genai.GenerativeModel("gemini-2.0-flash")
    response = model.generate_content([prompt, img])
    text = response.text.strip()
    accept = text.upper().startswith("ACCEPT")
    return accept, text


# ── YAML-Katalog ──────────────────────────────────────────────────────────────

def load_catalog() -> dict:
    if CATALOG.exists():
        with open(CATALOG, encoding="utf-8") as f:
            return yaml.safe_load(f) or {}
    return {}


def save_catalog(doc: dict) -> None:
    with open(CATALOG, "w", encoding="utf-8") as f:
        yaml.dump(doc, f, allow_unicode=True, default_flow_style=False, sort_keys=False)


def archive_existing(doc: dict) -> dict:
    """Markiert alle bestehenden Templates als superseded und verschiebt Bilder."""
    ARCHIVE_DIR.mkdir(parents=True, exist_ok=True)
    for t in doc.get("templates", []):
        if t.get("status") == "superseded":
            continue
        t["status"] = "superseded"
        t["enabled"] = False
        img_path = BASE_DIR / t.get("asset_path", "")
        if img_path.exists():
            dest = ARCHIVE_DIR / img_path.name
            shutil.move(str(img_path), str(dest))
            t["asset_path"] = f"assets/morphology/archive/{img_path.name}"
            logging.info(f"  Archiviert: {img_path.name}")
    return doc


def already_have(doc: dict, hubble_type: str) -> int:
    """Zählt bereits vorhandene (nicht-superseded) Einträge für einen Typ."""
    return sum(
        1 for t in doc.get("templates", [])
        if t.get("hubble_type") == hubble_type
        and t.get("status") != "superseded"
        and t.get("enabled", False)
    )


def save_image(img: Image.Image, hubble_type: str, galaxy_id: str) -> tuple[str, list[int]]:
    """Speichert Bild nach assets/morphology/{type}/. Gibt (asset_path, [w,h]) zurück."""
    safe_id = re.sub(r"[^A-Za-z0-9_-]", "_", galaxy_id.strip()).strip("_")
    type_dir = ASSETS_DIR / hubble_type
    type_dir.mkdir(parents=True, exist_ok=True)
    filename = f"{safe_id}.jpg"
    out_path = type_dir / filename

    if img.width > IMG_MAX_PX:
        h = int(IMG_MAX_PX * img.height / img.width)
        img = img.resize((IMG_MAX_PX, h), Image.LANCZOS)

    img.save(str(out_path), "JPEG", quality=90)
    return f"assets/morphology/{hubble_type}/{filename}", list(img.size)


def append_entry(doc: dict, entry: dict) -> dict:
    doc.setdefault("templates", []).append(entry)
    return doc


# ── SIMBAD Query ──────────────────────────────────────────────────────────────

def query_simbad(morph_pattern: str) -> list[dict]:
    """
    Fragt SIMBAD TAP nach Galaxien mit gegebenem morph_type LIKE-Pattern.
    Gibt nach angularer Größe absteigend sortierte Kandidatenliste zurück.
    """
    adql = f"""
        SELECT TOP {SIMBAD_LIMIT}
               main_id, ra, dec, morph_type, galdim_majaxis, galdim_minaxis
        FROM   basic
        WHERE  otype = 'G'
          AND  morph_type LIKE '{morph_pattern}'
          AND  galdim_majaxis BETWEEN {MIN_AXIS_ARCMIN} AND {MAX_AXIS_ARCMIN}
          AND  galdim_minaxis IS NOT NULL
          AND  galdim_minaxis > 0
        ORDER BY galdim_majaxis DESC
    """
    try:
        result = Simbad.query_tap(adql)
    except Exception as e:
        logging.error(f"SIMBAD TAP query failed: {e}")
        return []

    if result is None or len(result) == 0:
        return []

    rows = []
    for row in result:
        try:
            rows.append({
                "main_id":   str(row["main_id"]).strip(),
                "ra":        float(row["ra"]),
                "dec":       float(row["dec"]),
                "morph":     str(row["morph_type"]).strip(),
                "maj":       float(row["galdim_majaxis"]),
                "min_ax":    float(row["galdim_minaxis"]),
            })
        except (ValueError, TypeError):
            continue
    return rows


# ── Haupt-Scraping-Funktion ───────────────────────────────────────────────────

def scrape_type(hubble_type: str, dry_run: bool = False) -> int:
    """
    Scrapt Bilder für einen Hubble-Typ.
    Gibt Anzahl der erfolgreich heruntergeladenen Bilder zurück.
    """
    cfg = HUBBLE_TYPES[hubble_type]
    log = logging.getLogger(__name__)
    log.info(f"══ {hubble_type} ══")

    doc = load_catalog()
    have = already_have(doc, hubble_type)
    if have >= IMAGES_PER_TYPE:
        log.info(f"  Bereits {have} Bilder vorhanden — überspringe")
        return have

    need = IMAGES_PER_TYPE - have

    # ── SIMBAD Discovery ──────────────────────────────────────────────────────
    candidates = query_simbad(cfg["query"])
    log.info(f"  {len(candidates)} Kandidaten aus SIMBAD")

    downloaded = 0
    seen_ids: set[str] = set()

    for row in candidates:
        if downloaded >= need:
            break

        gal_id = row["main_id"]
        if gal_id in seen_ids:
            continue
        seen_ids.add(gal_id)

        morph = row["morph"]
        ra, dec = row["ra"], row["dec"]
        maj, min_ax = row["maj"], row["min_ax"]

        if maj <= 0:
            continue
        b_a = min_ax / maj

        # ── Achsenverhältnis-Filter ────────────────────────────────────────────
        if hubble_type.startswith("E"):
            b_a_min = cfg.get("b_a_min", 0.0)
            b_a_max = cfg.get("b_a_max", 1.01)
            if not (b_a_min <= b_a < b_a_max):
                continue
        elif cfg.get("face_on") and b_a < cfg.get("b_a_min", BA_FACE_ON):
            continue

        # ── RC3 Feinklassifikation ─────────────────────────────────────────────
        if not matches_hubble_type(morph, hubble_type):
            continue

        log.info(f"  Kandidat: {gal_id:25s} morph={morph:15s} b/a={b_a:.2f} maj={maj:.1f}'")

        if dry_run:
            downloaded += 1
            continue

        # ── Bild-Download ──────────────────────────────────────────────────────
        time.sleep(REQUEST_DELAY)
        img = download_sdss(ra, dec, maj)
        source = "SDSS DR17"
        if img is None:
            img = download_skyview(ra, dec, maj)
            source = "DSS2/STScI"
        if img is None:
            log.info(f"    Kein Bild verfügbar — überspringe")
            continue

        # ── Gemini Vision QA ───────────────────────────────────────────────────
        try:
            accept, notes = gemini_quality_check(img, hubble_type)
        except Exception as e:
            log.warning(f"    Gemini-Fehler: {e} — überspringe")
            continue

        if not accept:
            log.info(f"    Gemini REJECT: {notes[:80]}")
            continue

        # ── Speichern + YAML-Eintrag ───────────────────────────────────────────
        asset_path, resolution = save_image(img, hubble_type, gal_id)
        safe_id = re.sub(r"[^A-Za-z0-9_-]", "_", gal_id.strip()).strip("_")
        license_str = "CC-BY-4.0 (SDSS DR17)" if source == "SDSS DR17" else "Public Domain (DSS2/STScI)"
        credit_str  = "SDSS Collaboration / sdss.org" if source == "SDSS DR17" else "Digitized Sky Survey, STScI"

        entry = {
            "id":                f"{safe_id.lower()}_{hubble_type.lower().replace('+', 'p').replace('-', 'm')}",
            "enabled":           True,
            "status":            "available",
            "name":              gal_id,
            "designation":       gal_id,
            "hubble_type":       hubble_type,
            "hubble_description": f"De Vaucouleurs {hubble_type} — SIMBAD RC3 morph_type: {morph}",
            "file":              Path(asset_path).name,
            "asset_path":        asset_path,
            "source_archive":    source,
            "license":           license_str,
            "credit":            credit_str,
            "resolution_px":     resolution,
            "orientation":       "face-on" if cfg.get("face_on") else "varies",
            "morph_type_raw":    morph,
            "simbad_verified":   True,
            "quality_notes":     notes,
        }

        doc = load_catalog()
        doc = append_entry(doc, entry)
        save_catalog(doc)

        log.info(f"    ✓ {gal_id} → {asset_path}  [{notes[:60]}]")
        downloaded += 1

    if downloaded < need and not dry_run:
        log.warning(f"  Nur {downloaded}/{need} neue Bilder gefunden")

    return downloaded


# ── Einstiegspunkt ────────────────────────────────────────────────────────────

def main() -> None:
    parser = argparse.ArgumentParser(
        description="Galaxy Image Scraper (BL-03) — De-Vaucouleurs Morphologie-Templates"
    )
    parser.add_argument(
        "--type", nargs="+", metavar="TYPE",
        help="Zu verarbeitende Typen (z.B. Sb SBb E3); Standard: alle"
    )
    parser.add_argument(
        "--dry-run", action="store_true",
        help="Kein Download, nur Kandidaten anzeigen"
    )
    parser.add_argument(
        "--reset", action="store_true",
        help="Bestehende Katalog-Einträge archivieren und Bilder verschieben"
    )
    args = parser.parse_args()

    # Env laden: tools/.env oder Umgebungsvariable
    load_dotenv(TOOLS_DIR / ".env")

    api_key = os.environ.get("GEMINI_API_KEY")
    if not api_key and not args.dry_run:
        print("Fehler: GEMINI_API_KEY nicht gesetzt.", file=sys.stderr)
        print("  export GEMINI_API_KEY=... oder tools/.env befüllen.", file=sys.stderr)
        sys.exit(1)

    if api_key:
        genai.configure(api_key=api_key)

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s  %(message)s",
        datefmt="%H:%M:%S",
    )

    # Archivieren (--reset)
    if args.reset:
        logging.info("Archiviere bestehende Einträge …")
        doc = load_catalog()
        doc = archive_existing(doc)
        save_catalog(doc)
        logging.info("Archivierung abgeschlossen.")

    types_to_process = args.type if args.type else list(HUBBLE_TYPES.keys())

    # Unbekannte Typen abfangen
    unknown = [t for t in types_to_process if t not in HUBBLE_TYPES]
    if unknown:
        print(f"Unbekannte Typen: {', '.join(unknown)}", file=sys.stderr)
        print(f"Verfügbar: {', '.join(HUBBLE_TYPES.keys())}", file=sys.stderr)
        sys.exit(1)

    total = 0
    for t in types_to_process:
        count = scrape_type(t, dry_run=args.dry_run)
        total += count
        time.sleep(1.0)  # höfliche Pause zwischen Typen

    logging.info(f"Fertig. Gesamtbilder: {total}")


if __name__ == "__main__":
    main()
