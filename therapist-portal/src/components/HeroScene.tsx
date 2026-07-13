'use client';

import { useEffect, useRef } from 'react';
import * as THREE from 'three';

/**
 * HeroScene — "from noise to clarity", the Ode story in 3D.
 *
 * A cloud of thought-particles cycles through an emotional arc:
 *   chaos    — dispersed, jittery, cold blue-grey (the overwhelmed mind)
 *   gather   — particles are drawn together, warming from cold to gold
 *   clarity  — a calm, breathing constellation sphere with its
 *              connections lit (patterns found), then a gentle release
 * Faint emotion words orbit the scene: heavy ones during chaos, lighter
 * ones once the constellation forms — echoing the app's emotional-tone
 * analysis. The whole structure spins and steers toward the cursor.
 *
 * Usability guarantees:
 *  - pointer-events: none — never intercepts clicks or scroll
 *  - pauses when the hero scrolls out of view (IntersectionObserver)
 *  - prefers-reduced-motion → renders one static "clarity" frame, no loop
 *  - DPR capped at 2, particle counts reduced on small screens
 *  - full disposal on unmount; silently renders nothing if WebGL fails
 */

const POINT_VERTEX = /* glsl */ `
  attribute vec3 aChaos;
  attribute float aPhase;
  attribute float aScale;
  attribute float aMix;
  uniform float uTime;
  uniform float uPixelRatio;
  uniform float uMorph;
  varying float vMix;
  varying float vAlpha;

  void main() {
    // morph between the scattered "noisy mind" and the ordered sphere
    vec3 pos = mix(aChaos, position, uMorph);
    float loose = 1.0 - uMorph;
    pos += vec3(
      sin(uTime * 1.6 + aPhase * 4.1),
      cos(uTime * 1.2 + aPhase * 6.3),
      sin(uTime * 0.9 + aPhase * 2.7)
    ) * (0.12 + loose * 1.7);

    vec4 mv = modelViewMatrix * vec4(pos, 1.0);
    gl_Position = projectionMatrix * mv;
    gl_PointSize = aScale * uPixelRatio * (160.0 / -mv.z);

    float twinkle = 0.65 + 0.35 * sin(uTime * (0.8 + fract(aPhase) * 1.4) + aPhase * 6.2831);
    float depth = clamp((-mv.z - 12.0) / 26.0, 0.0, 1.0);
    // slightly dimmer while scattered, bright once gathered
    vAlpha = twinkle * mix(1.15, 0.18, depth) * mix(0.85, 1.0, uMorph);
    vMix = aMix;
  }
`;

const DUST_VERTEX = /* glsl */ `
  attribute float aPhase;
  attribute float aScale;
  attribute float aSpeed;
  attribute float aMix;
  uniform float uTime;
  uniform float uPixelRatio;
  varying float vMix;
  varying float vAlpha;

  void main() {
    vec3 pos = position;
    // drift sideways AND toward the camera, wrapping — a fly-through depth cue
    pos.x = mod(pos.x + uTime * aSpeed * 0.6 + 32.0, 64.0) - 32.0;
    pos.z = mod(pos.z + uTime * aSpeed * 1.6 + 22.0, 28.0) - 24.0;
    pos.y += sin(pos.x * 0.2 + uTime * 0.3 + aPhase) * 0.9;

    vec4 mv = modelViewMatrix * vec4(pos, 1.0);
    gl_Position = projectionMatrix * mv;
    gl_PointSize = aScale * uPixelRatio * (170.0 / -mv.z);

    float twinkle = 0.7 + 0.3 * sin(uTime * (0.6 + aSpeed) + aPhase * 3.0);
    float depth = clamp((-mv.z - 6.0) / 34.0, 0.0, 1.0);
    // fade out before a particle reaches the camera so it never "pops"
    float nearFade = smoothstep(3.5, 9.0, -mv.z);
    vAlpha = twinkle * mix(1.0, 0.3, depth) * nearFade;
    vMix = aMix;
  }
`;

const POINT_FRAGMENT = /* glsl */ `
  uniform vec3 uColorA;
  uniform vec3 uColorB;
  uniform vec3 uColorC;
  uniform vec3 uColdA;
  uniform vec3 uColdB;
  uniform float uWarmth;
  varying float vMix;
  varying float vAlpha;

  void main() {
    vec2 uv = gl_PointCoord - 0.5;
    float d = length(uv);
    if (d > 0.5) discard;
    float glow = pow(1.0 - d * 2.0, 2.2);
    vec3 warm = mix(uColorA, uColorB, smoothstep(0.6, 0.95, vMix));
    warm = mix(warm, uColorC, step(0.97, vMix));
    vec3 cold = mix(uColdA, uColdB, smoothstep(0.4, 0.9, vMix));
    // the emotional temperature of the whole scene
    vec3 col = mix(cold, warm, uWarmth);
    gl_FragColor = vec4(col, glow * vAlpha);
  }
`;

function makePointMaterial(vertexShader: string) {
  return new THREE.ShaderMaterial({
    vertexShader,
    fragmentShader: POINT_FRAGMENT,
    transparent: true,
    depthWrite: false,
    depthTest: false,
    blending: THREE.AdditiveBlending,
    uniforms: {
      uTime: { value: 0 },
      uPixelRatio: { value: Math.min(window.devicePixelRatio, 2) },
      uMorph: { value: 1 },
      uWarmth: { value: 1 },
      uColorA: { value: new THREE.Color('#d4a56a') }, // gold
      uColorB: { value: new THREE.Color('#5b8db8') }, // dusk blue
      uColorC: { value: new THREE.Color('#f0e6d8') }, // rare cream sparks
      uColdA: { value: new THREE.Color('#7d95b5') }, // muted slate blue
      uColdB: { value: new THREE.Color('#9aa3b3') }, // grey mist
    },
  });
}

// serif emotion word rendered onto a sprite
function makeWordSprite(text: string, warm: boolean): THREE.Sprite {
  const canvas = document.createElement('canvas');
  canvas.width = 512;
  canvas.height = 128;
  const ctx = canvas.getContext('2d');
  if (ctx) {
    ctx.font = 'italic 300 58px "Cormorant Garamond", Georgia, serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillStyle = warm ? 'rgba(232,221,208,0.95)' : 'rgba(150,160,178,0.9)';
    ctx.fillText(text, 256, 68);
  }
  const tex = new THREE.CanvasTexture(canvas);
  const mat = new THREE.SpriteMaterial({ map: tex, transparent: true, opacity: 0, depthWrite: false });
  const sprite = new THREE.Sprite(mat);
  sprite.scale.set(6.4, 1.6, 1);
  return sprite;
}

// The field holds its formed "clarity" state permanently — no chaos↔clarity
// cycle. The scene should read as a calm constellation with slow drift, not a
// performance competing with the headline.
const MOOD = 1;

export default function HeroScene() {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    let renderer: THREE.WebGLRenderer;
    try {
      renderer = new THREE.WebGLRenderer({ alpha: true, antialias: false });
    } catch {
      return; // no WebGL — hero simply has no scene
    }

    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const isSmall = window.innerWidth < 960; // below this the phone mockup is hidden
    const SHELL_COUNT = isSmall ? 620 : 1150;
    const INNER_COUNT = isSmall ? 220 : 420;
    const RING_COUNT = isSmall ? 160 : 280;
    const DUST_COUNT = isSmall ? 420 : 1100;
    const MAX_SEGMENTS = isSmall ? 500 : 1000;
    const RADIUS = isSmall ? 7.5 : 9.5;

    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    renderer.setClearColor(0x000000, 0);
    renderer.domElement.style.position = 'absolute';
    renderer.domElement.style.inset = '0';
    renderer.domElement.style.width = '100%';
    renderer.domElement.style.height = '100%';
    renderer.domElement.style.opacity = '0';
    renderer.domElement.style.transition = 'opacity 1.6s ease';
    container.appendChild(renderer.domElement);

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(55, 1, 0.1, 100);
    camera.position.set(0, 0, 26);

    const disposables: { dispose: () => void }[] = [];

    /* ── Thought constellation ─────────────────────────────────────── */
    const orbGroup = new THREE.Group();
    // desktop: behind the phone mockup; mobile: low in the frame like a
    // rising moon, so the glow never sits behind the headline or body copy
    orbGroup.position.set(isSmall ? 0 : 7.5, isSmall ? -7.5 : 0.5, isSmall ? -9 : -6);
    scene.add(orbGroup);

    // scattered "chaos" position for a particle — a wide, flattened cloud
    // kept shallow in z so the restless mind stays clearly visible on screen
    const chaosOf = () => {
      const dir = new THREE.Vector3().randomDirection();
      const r = RADIUS * (1.0 + Math.random() * 1.2);
      return [dir.x * r * 1.5, dir.y * r * 0.9, dir.z * r * 0.6];
    };

    const orbTotal = SHELL_COUNT + INNER_COUNT;
    const orbPos = new Float32Array(orbTotal * 3);
    const orbChaos = new Float32Array(orbTotal * 3);
    const orbPhase = new Float32Array(orbTotal);
    const orbScale = new Float32Array(orbTotal);
    const orbMix = new Float32Array(orbTotal);

    // shell: fibonacci sphere with slight jitter → even, organic coverage
    const GA = Math.PI * (3 - Math.sqrt(5));
    for (let i = 0; i < SHELL_COUNT; i++) {
      const y = 1 - (i / (SHELL_COUNT - 1)) * 2;
      const r = Math.sqrt(1 - y * y);
      const theta = GA * i;
      const jitter = 0.94 + Math.random() * 0.1;
      orbPos[i * 3] = Math.cos(theta) * r * RADIUS * jitter;
      orbPos[i * 3 + 1] = y * RADIUS * jitter;
      orbPos[i * 3 + 2] = Math.sin(theta) * r * RADIUS * jitter;
      [orbChaos[i * 3], orbChaos[i * 3 + 1], orbChaos[i * 3 + 2]] = chaosOf();
      orbPhase[i] = Math.random() * Math.PI * 2;
      orbScale[i] = 0.8 + Math.random() * 1.3;
      orbMix[i] = Math.random();
    }
    // interior: volumetric fill so the sphere reads as a body, not a hollow shell
    for (let i = SHELL_COUNT; i < orbTotal; i++) {
      const dir = new THREE.Vector3().randomDirection();
      const r = RADIUS * 0.92 * Math.cbrt(Math.random());
      orbPos[i * 3] = dir.x * r;
      orbPos[i * 3 + 1] = dir.y * r;
      orbPos[i * 3 + 2] = dir.z * r;
      [orbChaos[i * 3], orbChaos[i * 3 + 1], orbChaos[i * 3 + 2]] = chaosOf();
      orbPhase[i] = Math.random() * Math.PI * 2;
      orbScale[i] = 0.45 + Math.random() * 0.8;
      orbMix[i] = Math.random();
    }

    const orbGeom = new THREE.BufferGeometry();
    orbGeom.setAttribute('position', new THREE.BufferAttribute(orbPos, 3));
    orbGeom.setAttribute('aChaos', new THREE.BufferAttribute(orbChaos, 3));
    orbGeom.setAttribute('aPhase', new THREE.BufferAttribute(orbPhase, 1));
    orbGeom.setAttribute('aScale', new THREE.BufferAttribute(orbScale, 1));
    orbGeom.setAttribute('aMix', new THREE.BufferAttribute(orbMix, 1));
    const orbMat = makePointMaterial(POINT_VERTEX);
    orbGroup.add(new THREE.Points(orbGeom, orbMat));
    disposables.push(orbGeom, orbMat);

    // constellation lines between nearby shell points — "connecting the dots"
    const threshold = RADIUS * 0.24;
    const linePositions: number[] = [];
    let segments = 0;
    outer: for (let i = 0; i < SHELL_COUNT; i++) {
      for (let j = i + 1; j < SHELL_COUNT; j++) {
        const dx = orbPos[i * 3] - orbPos[j * 3];
        const dy = orbPos[i * 3 + 1] - orbPos[j * 3 + 1];
        const dz = orbPos[i * 3 + 2] - orbPos[j * 3 + 2];
        if (dx * dx + dy * dy + dz * dz < threshold * threshold && Math.random() < 0.5) {
          linePositions.push(
            orbPos[i * 3], orbPos[i * 3 + 1], orbPos[i * 3 + 2],
            orbPos[j * 3], orbPos[j * 3 + 1], orbPos[j * 3 + 2],
          );
          if (++segments >= MAX_SEGMENTS) break outer;
        }
      }
    }
    const lineGeom = new THREE.BufferGeometry();
    lineGeom.setAttribute('position', new THREE.BufferAttribute(new Float32Array(linePositions), 3));
    const lineMat = new THREE.LineBasicMaterial({
      color: new THREE.Color('#c8955a'),
      transparent: true,
      opacity: 0,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
    });
    orbGroup.add(new THREE.LineSegments(lineGeom, lineMat));
    disposables.push(lineGeom, lineMat);

    // orbiting ring — a tilted halo that makes the rotation obvious
    const ringGroup = new THREE.Group();
    ringGroup.rotation.set(1.9, 0, 0.35);
    orbGroup.add(ringGroup);
    const ringPos = new Float32Array(RING_COUNT * 3);
    const ringChaos = new Float32Array(RING_COUNT * 3);
    const ringPhase = new Float32Array(RING_COUNT);
    const ringScale = new Float32Array(RING_COUNT);
    const ringMix = new Float32Array(RING_COUNT);
    for (let i = 0; i < RING_COUNT; i++) {
      const a = (i / RING_COUNT) * Math.PI * 2;
      const rr = RADIUS * (1.45 + (Math.random() - 0.5) * 0.14);
      ringPos[i * 3] = Math.cos(a) * rr;
      ringPos[i * 3 + 1] = (Math.random() - 0.5) * 0.5;
      ringPos[i * 3 + 2] = Math.sin(a) * rr;
      [ringChaos[i * 3], ringChaos[i * 3 + 1], ringChaos[i * 3 + 2]] = chaosOf();
      ringPhase[i] = Math.random() * Math.PI * 2;
      ringScale[i] = 0.5 + Math.random() * 1.0;
      ringMix[i] = Math.random() * 0.7;
    }
    const ringGeom = new THREE.BufferGeometry();
    ringGeom.setAttribute('position', new THREE.BufferAttribute(ringPos, 3));
    ringGeom.setAttribute('aChaos', new THREE.BufferAttribute(ringChaos, 3));
    ringGeom.setAttribute('aPhase', new THREE.BufferAttribute(ringPhase, 1));
    ringGeom.setAttribute('aScale', new THREE.BufferAttribute(ringScale, 1));
    ringGeom.setAttribute('aMix', new THREE.BufferAttribute(ringMix, 1));
    const ringMat = makePointMaterial(POINT_VERTEX);
    ringGroup.add(new THREE.Points(ringGeom, ringMat));
    disposables.push(ringGeom, ringMat);

    // fresnel-rimmed inner sphere — a glowing silhouette edge that makes the
    // orb read as a solid 3D body; only visible once the mind has "settled"
    const fresnelMat = new THREE.ShaderMaterial({
      vertexShader: /* glsl */ `
        varying vec3 vNormal;
        varying vec3 vView;
        void main() {
          vNormal = normalize(normalMatrix * normal);
          vec4 mv = modelViewMatrix * vec4(position, 1.0);
          vView = -mv.xyz;
          gl_Position = projectionMatrix * mv;
        }
      `,
      fragmentShader: /* glsl */ `
        uniform vec3 uColor;
        uniform float uTime;
        uniform float uMorph;
        varying vec3 vNormal;
        varying vec3 vView;
        void main() {
          float rim = pow(1.0 - abs(dot(normalize(vView), normalize(vNormal))), 2.6);
          float pulse = 0.85 + 0.15 * sin(uTime * 0.7);
          gl_FragColor = vec4(uColor, rim * 0.5 * pulse * uMorph);
        }
      `,
      uniforms: {
        uColor: { value: new THREE.Color('#d4a56a') },
        uTime: { value: 0 },
        uMorph: { value: 1 },
      },
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
    });
    const fresnelGeom = new THREE.SphereGeometry(RADIUS * 0.97, 48, 48);
    orbGroup.add(new THREE.Mesh(fresnelGeom, fresnelMat));
    disposables.push(fresnelGeom, fresnelMat);

    // soft nucleus glow at the orb's heart
    const glowCanvas = document.createElement('canvas');
    glowCanvas.width = glowCanvas.height = 128;
    const gctx = glowCanvas.getContext('2d');
    if (gctx) {
      const grad = gctx.createRadialGradient(64, 64, 0, 64, 64, 64);
      grad.addColorStop(0, 'rgba(212,165,106,0.55)');
      grad.addColorStop(0.35, 'rgba(200,149,90,0.18)');
      grad.addColorStop(1, 'rgba(200,149,90,0)');
      gctx.fillStyle = grad;
      gctx.fillRect(0, 0, 128, 128);
    }
    const glowTex = new THREE.CanvasTexture(glowCanvas);
    const glowMat = new THREE.SpriteMaterial({
      map: glowTex, transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
    });
    const glowSprite = new THREE.Sprite(glowMat);
    glowSprite.scale.setScalar(RADIUS * 1.6);
    orbGroup.add(glowSprite);
    disposables.push(glowTex, glowMat);

    /* ── Emotion words ─────────────────────────────────────────────── */
    // heavy words drift in the chaos; lighter ones orbit the formed sphere
    const heavyWords = ['anxious', 'tired', 'overwhelmed'];
    const lightWords = ['calmer', 'hopeful', 'grateful'];
    const wordGroup = new THREE.Group();
    wordGroup.position.copy(orbGroup.position);
    scene.add(wordGroup);

    type WordEntry = { sprite: THREE.Sprite; angle: number; height: number; radius: number; speed: number; warm: boolean };
    const words: WordEntry[] = [];
    // heavy/cold words belonged to the chaos phase — with the field held in
    // clarity they would never be visible, so they are not created at all
    void heavyWords;
    lightWords.forEach((w, i) => {
      const sprite = makeWordSprite(w, true);
      wordGroup.add(sprite);
      disposables.push(sprite.material.map!, sprite.material);
      words.push({
        sprite, warm: true,
        angle: (i / lightWords.length) * Math.PI * 2,
        height: [3.8, -4.2, 6.2][i] ?? 0,
        radius: RADIUS * 1.75,
        speed: 0.05 + i * 0.01,
      });
    });

    /* ── Ambient dust flying past the camera ───────────────────────── */
    const dustPos = new Float32Array(DUST_COUNT * 3);
    const dustPhase = new Float32Array(DUST_COUNT);
    const dustScale = new Float32Array(DUST_COUNT);
    const dustSpeed = new Float32Array(DUST_COUNT);
    const dustMix = new Float32Array(DUST_COUNT);
    for (let i = 0; i < DUST_COUNT; i++) {
      dustPos[i * 3] = (Math.random() - 0.5) * 64;
      dustPos[i * 3 + 1] = (Math.random() - 0.5) * 26;
      dustPos[i * 3 + 2] = -24 + Math.random() * 28;
      dustPhase[i] = Math.random() * Math.PI * 2;
      dustScale[i] = 0.4 + Math.random() * 0.9;
      dustSpeed[i] = 0.15 + Math.random() * 0.45; // slow ambient drift, not a fly-through
      dustMix[i] = Math.random();
    }
    const dustGeom = new THREE.BufferGeometry();
    dustGeom.setAttribute('position', new THREE.BufferAttribute(dustPos, 3));
    dustGeom.setAttribute('aPhase', new THREE.BufferAttribute(dustPhase, 1));
    dustGeom.setAttribute('aScale', new THREE.BufferAttribute(dustScale, 1));
    dustGeom.setAttribute('aSpeed', new THREE.BufferAttribute(dustSpeed, 1));
    dustGeom.setAttribute('aMix', new THREE.BufferAttribute(dustMix, 1));
    const dustMat = makePointMaterial(DUST_VERTEX);
    scene.add(new THREE.Points(dustGeom, dustMat));
    disposables.push(dustGeom, dustMat);

    /* ── Sizing ────────────────────────────────────────────────────── */
    const resize = () => {
      const { clientWidth: w, clientHeight: h } = container;
      if (!w || !h) return;
      renderer.setSize(w, h, false);
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
    };
    resize();
    const ro = new ResizeObserver(resize);
    ro.observe(container);

    /* ── Interaction: cursor steers the structure, scroll tilts it ─── */
    const mouse = { x: 0, y: 0 };
    const onPointerMove = (e: PointerEvent) => {
      mouse.x = (e.clientX / window.innerWidth) * 2 - 1;
      mouse.y = (e.clientY / window.innerHeight) * 2 - 1;
    };
    let scrollTilt = 0;
    const onScroll = () => {
      scrollTilt = Math.min(window.scrollY / window.innerHeight, 1.2);
    };
    const interactive = !reducedMotion;
    if (interactive) {
      window.addEventListener('pointermove', onPointerMove, { passive: true });
      window.addEventListener('scroll', onScroll, { passive: true });
    }

    /* ── Render loop, paused when hero is off-screen ───────────────── */
    const clock = new THREE.Clock();
    let elapsed = 4.2; // start just before the first gathering
    let raf = 0;
    let inView = true;
    let firstFrame = true;
    let spin = 0;

    const renderFrame = () => {
      const dt = clock.getDelta();
      elapsed += dt;
      spin += dt * 0.025; // barely-perceptible drift, not a visible rotation

      const mood = MOOD;

      orbMat.uniforms.uTime.value = elapsed;
      ringMat.uniforms.uTime.value = elapsed;
      dustMat.uniforms.uTime.value = elapsed;
      fresnelMat.uniforms.uTime.value = elapsed;
      orbMat.uniforms.uMorph.value = mood;
      ringMat.uniforms.uMorph.value = mood;
      fresnelMat.uniforms.uMorph.value = mood;
      orbMat.uniforms.uWarmth.value = mood;
      ringMat.uniforms.uWarmth.value = mood;
      dustMat.uniforms.uWarmth.value = 0.4 + mood * 0.6; // dust never goes fully cold
      lineMat.opacity = 0.16 * mood;
      glowMat.opacity = mood * (isSmall ? 0.55 : 1);

      // orb follows the cursor with gentle inertia + scroll adds a slight tilt
      const targetY = spin + mouse.x * 0.3;
      const targetX = 0.18 + mouse.y * 0.15 + scrollTilt * 0.3;
      orbGroup.rotation.y += (targetY - orbGroup.rotation.y) * 0.04;
      orbGroup.rotation.x += (targetX - orbGroup.rotation.x) * 0.04;
      // ring drifts slowly around the orb
      ringGroup.rotation.y -= dt * 0.06;
      // nucleus breathes, softly
      glowSprite.scale.setScalar(RADIUS * (1.55 + Math.sin(elapsed * 0.35) * 0.04));

      // emotion words: with the field held in clarity only the light ones show
      for (const w of words) {
        w.angle += dt * w.speed;
        const bob = Math.sin(elapsed * 0.3 + w.height) * 0.25;
        w.sprite.position.set(
          Math.cos(w.angle) * w.radius,
          w.height + bob,
          Math.sin(w.angle) * w.radius,
        );
        let presence = w.warm ? mood : 1 - mood;
        // fade words out as they swing toward the headline/copy column
        const worldX = wordGroup.position.x + Math.cos(w.angle) * w.radius;
        presence *= THREE.MathUtils.smoothstep(worldX, -1, 4);
        // keep words behind-the-content quiet: cap opacity low
        (w.sprite.material as THREE.SpriteMaterial).opacity =
          presence * (0.3 + 0.08 * Math.sin(elapsed * 0.8 + w.angle * 3));
      }

      // camera parallax opposite the cursor deepens the perspective — subtle
      camera.position.x += (mouse.x * 0.8 - camera.position.x) * 0.03;
      camera.position.y += (-mouse.y * 0.5 - camera.position.y) * 0.03;
      camera.lookAt(isSmall ? 0 : 3.5, 0, 0);

      renderer.render(scene, camera);
      if (firstFrame) {
        firstFrame = false;
        renderer.domElement.style.opacity = '1';
      }
    };

    const loop = () => {
      renderFrame();
      raf = requestAnimationFrame(loop);
    };

    const io = new IntersectionObserver(
      ([entry]) => {
        const nowInView = entry.isIntersecting;
        if (nowInView === inView) return;
        inView = nowInView;
        if (reducedMotion) return;
        if (inView) {
          clock.getDelta(); // discard time spent off-screen
          raf = requestAnimationFrame(loop);
        } else {
          cancelAnimationFrame(raf);
        }
      },
      { threshold: 0 }
    );
    io.observe(container);

    if (reducedMotion) {
      elapsed = 14; // fully-formed, calm constellation
      spin = 0.6;
      renderFrame(); // single static frame
    } else {
      raf = requestAnimationFrame(loop);
    }

    return () => {
      cancelAnimationFrame(raf);
      io.disconnect();
      ro.disconnect();
      if (interactive) {
        window.removeEventListener('pointermove', onPointerMove);
        window.removeEventListener('scroll', onScroll);
      }
      disposables.forEach(d => d.dispose());
      renderer.dispose();
      renderer.domElement.remove();
    };
  }, []);

  return (
    <div
      ref={containerRef}
      aria-hidden="true"
      style={{
        position: 'absolute',
        inset: 0,
        pointerEvents: 'none',
        // fade the field out toward the content below the hero
        maskImage: 'linear-gradient(180deg, transparent 0%, black 12%, black 80%, transparent 100%)',
        WebkitMaskImage: 'linear-gradient(180deg, transparent 0%, black 12%, black 80%, transparent 100%)',
      }}
    />
  );
}
