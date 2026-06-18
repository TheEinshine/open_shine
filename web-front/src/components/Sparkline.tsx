interface Props {
  data: number[];
  color?: string;
  height?: number;
  max?: number;
}

// Sparkline draws a tiny filled line chart as inline SVG (no chart library).
export function Sparkline({ data, color = "var(--accent)", height = 44, max = 100 }: Props) {
  const width = 100; // viewBox units; scales to container via width:100%
  if (data.length < 2) {
    return <div className="muted mono" style={{ fontSize: 11, height }}>not enough data yet</div>;
  }
  const top = Math.max(max, ...data) || 1;
  const step = width / (data.length - 1);
  const points = data.map((v, i) => {
    const x = i * step;
    const y = height - (Math.max(0, v) / top) * height;
    return [x, y] as const;
  });
  const line = points.map(([x, y]) => `${x.toFixed(2)},${y.toFixed(2)}`).join(" ");
  const area = `0,${height} ${line} ${width},${height}`;
  const gid = `g${Math.round(points[0][1])}-${data.length}`;

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      preserveAspectRatio="none"
      width="100%"
      height={height}
      style={{ display: "block" }}
    >
      <defs>
        <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity="0.28" />
          <stop offset="100%" stopColor={color} stopOpacity="0" />
        </linearGradient>
      </defs>
      <polygon points={area} fill={`url(#${gid})`} />
      <polyline points={line} fill="none" stroke={color} strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}
