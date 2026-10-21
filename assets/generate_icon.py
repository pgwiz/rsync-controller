from PIL import Image, ImageDraw, ImageFont
import os

def create_rp_icon(size=512):
    # Create RGBA canvas
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # Rounded rectangle coordinates
    padding = size * 0.05
    rect = [padding, padding, size - padding, size - padding]
    corner_radius = size * 0.18

    # Background: Matte obsidian black with subtle dark border
    bg_color = (14, 14, 18, 255)
    border_color = (48, 48, 54, 255)
    border_width = max(2, int(size * 0.03))

    # Outer border
    draw.rounded_rectangle(rect, radius=corner_radius, fill=bg_color, outline=border_color, width=border_width)

    # Subtle inner gloss / bevel at top edge
    inner_padding = padding + border_width
    inner_rect = [inner_padding + 2, inner_padding + 2, size - inner_padding - 2, (size / 2)]
    # We can draw subtle highlight line
    draw.line(
        [(padding + corner_radius, inner_padding + 1), (size - padding - corner_radius, inner_padding + 1)],
        fill=(80, 80, 90, 180),
        width=max(1, int(size * 0.015))
    )

    # Typography: "rp" in bold lowercase
    # Load Segoe UI Bold or Arial Bold
    font_paths = [
        r"C:\Windows\Fonts\segoeuib.ttf",
        r"C:\Windows\Fonts\arialbd.ttf",
        r"C:\Windows\Fonts\tahomabd.ttf"
    ]
    font_path = None
    for p in font_paths:
        if os.path.exists(p):
            font_path = p
            break

    # Font size relative to icon - user requested: "written rp in small black bg white text"
    # "in small" -> not gigantic filling 100%, but a crisp proportional badge (around 42% height)
    target_font_size = int(size * 0.42)
    font = ImageFont.truetype(font_path, target_font_size) if font_path else ImageFont.load_default()

    text = "rp"
    # Calculate text bounding box to center it
    bbox = draw.textbbox((0, 0), text, font=font)
    text_w = bbox[2] - bbox[0]
    text_h = bbox[3] - bbox[1]

    # Center position with slight visual offset for baseline
    text_x = (size - text_w) / 2 - bbox[0]
    # In fonts, baseline is lower, adjust y slightly
    text_y = (size - text_h) / 2 - bbox[1] - (size * 0.02)

    # Text color: Pure crisp white
    draw.text((text_x, text_y), text, font=font, fill=(255, 255, 255, 255))

    return img

def main():
    base_dir = r"E:\Backup\pgwiz\rsync"
    assets_dir = os.path.join(base_dir, "assets")
    ui_assets_dir = os.path.join(base_dir, "ui", "assets")

    # Generate master 512x512
    master = create_rp_icon(512)
    master_png = os.path.join(assets_dir, "icon.png")
    master.save(master_png, "PNG")
    print(f"[OK] Generated {master_png}")

    # UI PNG (64x64) and UI 192x192
    ui_64 = master.resize((64, 64), Image.Resampling.LANCZOS)
    ui_64.save(os.path.join(ui_assets_dir, "icon.png"), "PNG")
    ui_192 = master.resize((192, 192), Image.Resampling.LANCZOS)
    ui_192.save(os.path.join(ui_assets_dir, "icon-192.png"), "PNG")
    print(f"[OK] Generated UI PNG icons")

    # Windows ICO with standard mip levels: 256, 128, 64, 48, 32, 24, 16
    ico_sizes = [(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (24, 24), (16, 16)]
    ico_path = os.path.join(assets_dir, "icon.ico")
    master.save(ico_path, format="ICO", sizes=ico_sizes)
    print(f"[OK] Generated {ico_path} with sizes {ico_sizes}")

if __name__ == "__main__":
    main()
