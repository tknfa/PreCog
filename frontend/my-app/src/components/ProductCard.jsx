import { useEffect, useState } from "react";

function buildFallbackImage(name) {
  const label = encodeURIComponent((name || "Recommended Item").slice(0, 32));
  return `data:image/svg+xml;utf8,\
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 320 220">\
<defs>\
<linearGradient id="bg" x1="0" x2="1" y1="0" y2="1">\
<stop offset="0%" stop-color="%23111827"/>\
<stop offset="100%" stop-color="%23334155"/>\
</linearGradient>\
</defs>\
<rect width="320" height="220" rx="24" fill="url(%23bg)"/>\
<circle cx="84" cy="82" r="30" fill="%2360a5fa" opacity="0.85"/>\
<rect x="132" y="60" width="112" height="14" rx="7" fill="%23e5e7eb" opacity="0.95"/>\
<rect x="132" y="86" width="72" height="12" rx="6" fill="%23cbd5e1" opacity="0.85"/>\
<rect x="44" y="144" width="232" height="40" rx="12" fill="%230f172a" opacity="0.8"/>\
<text x="160" y="168" font-family="Arial, Helvetica, sans-serif" font-size="18" fill="white" text-anchor="middle">${label}</text>\
</svg>`;
}

export default function ProductCard({ name, img, imageUrl, price, isBookmarked, onBookmark }) {
  const fallbackImage = buildFallbackImage(name);
  const [resolvedImage, setResolvedImage] = useState(imageUrl || img || fallbackImage);

  useEffect(() => {
    setResolvedImage(imageUrl || img || fallbackImage);
  }, [fallbackImage, imageUrl, img]);

  return (
    <div className="productCard">
      <img
        src={resolvedImage}
        alt={name || "Recommended product"}
        className="product-img"
        loading="lazy"
        referrerPolicy="no-referrer"
        onError={() => setResolvedImage(fallbackImage)}
      />
      <h3>{name}</h3>
      <p>{price}</p>

      <button className="add-cart" onClick={onBookmark}>
        {isBookmarked ? "Remove" : "Bookmark"}
      </button>
    </div>
  );
}
