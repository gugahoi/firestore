export const meta = {
  name: 'pr-review',
  description: 'Fan out a multi-dimension review of the branch diff, adversarially verify each finding, report only survivors',
  whenToUse: 'Before opening/merging a PR: a stronger gate than a single inline review pass. Pass args.base to change the compare ref (default "main"); args.thorough=true uses a 3-skeptic majority panel.',
  phases: [
    { title: 'Review', detail: 'one reviewer per dimension, in parallel, each returns structured findings' },
    { title: 'Verify', detail: 'each finding faces skeptics prompted to refute it; only survivors are kept' },
  ],
}

// ---- knobs (parameterized via the `args` global passed to Workflow) ----
const BASE = (args && args.base) || 'main'
const SKEPTICS = args && args.thorough ? 3 : 1           // majority panel when thorough
const NEEDED = Math.floor(SKEPTICS / 2) + 1              // votes required to KILL a finding

// ---- the review dimensions (tuned to this Go/Firestore repo) ----
const DIMENSIONS = [
  { key: 'correctness', focus: 'logic bugs, nil/error-path mistakes, off-by-one, incorrect path normalization, wrong copy/move semantics (copy=Get+Set, move=copy+delete)' },
  { key: 'overwrite-safety', focus: 'does a copy/duplicate silently overwrite an existing document? Create-vs-Set semantics, missing existence checks, destructive defaults' },
  { key: 'error-handling', focus: 'silent failures, swallowed errors, errors not wrapped with %w, ignored return values, missing context propagation' },
  { key: 'conventions', focus: 'CLAUDE.md style: gofmt, context via cmd.Context(), typed context keys, JSON encoder settings, cobra Args validation, package/naming rules' },
]

// ---- schemas: the contract each agent must satisfy (and the prompt that shapes its thinking) ----
const FINDINGS = {
  type: 'object',
  additionalProperties: false,
  properties: {
    dimension: { type: 'string' },
    findings: {
      type: 'array',
      items: {
        type: 'object',
        additionalProperties: false,
        properties: {
          file: { type: 'string', description: 'path:line' },
          severity: { type: 'string', enum: ['low', 'medium', 'high', 'critical'] },
          severityRationale: { type: 'string', description: 'why this severity and not one lower' },
          title: { type: 'string' },
          explanation: { type: 'string', description: 'the concrete bug/issue and why it matters' },
          suggestedFix: { type: 'string' },
        },
        required: ['file', 'severity', 'severityRationale', 'title', 'explanation', 'suggestedFix'],
      },
    },
  },
  required: ['dimension', 'findings'],
}

const VERDICT = {
  type: 'object',
  additionalProperties: false,
  properties: {
    whyItMightBeFalse: { type: 'string', description: 'the strongest argument that this finding is NOT a real problem' },
    refuted: { type: 'boolean', description: 'true if the finding does not hold up under scrutiny' },
    confidence: { type: 'string', enum: ['low', 'medium', 'high'] },
  },
  required: ['whyItMightBeFalse', 'refuted', 'confidence'],
}

// ---- retry helper: agent() returns null on a terminal API failure (e.g. stream timeout).
// A fresh agent often survives a transient flake, so re-spawn before giving up. ----
const REVIEW_ATTEMPTS = 2
async function reviewWithRetry(d) {
  for (let i = 0; i < REVIEW_ATTEMPTS; i++) {
    const r = await agent(
      `Review the diff \`git diff ${BASE}...HEAD\` in this Go Firestore CLI repo. Run that git command yourself and read the changed files for context. Focus ONLY on: ${d.focus}. Report real, actionable findings — not nitpicks. If nothing, return an empty findings array.`,
      { label: `review:${d.key}${i ? ` (retry ${i})` : ''}`, phase: 'Review', schema: FINDINGS }
    )
    if (r) return r
    log(`review:${d.key} returned null on attempt ${i + 1}/${REVIEW_ATTEMPTS}${i + 1 < REVIEW_ATTEMPTS ? ' — retrying' : ' — giving up'}`)
  }
  return null
}

// ---- pipeline: each dimension verifies as soon as ITS review returns (no barrier) ----
const reviewed = await pipeline(
  DIMENSIONS,
  // stage 1 — review one dimension (with retry). Always return a wrapper so a failed
  // dimension stays observable downstream instead of collapsing to an indistinguishable [].
  async d => ({ dimension: d.key, review: await reviewWithRetry(d) }),
  // stage 2 — adversarially verify each finding; carry the failed-review signal through.
  (res, d) => parallel(
    (res.review?.findings || []).map(f => () =>
      parallel(Array.from({ length: SKEPTICS }, (_, i) => () =>
        agent(
          `You are a skeptical reviewer. A colleague claims this issue in \`${f.file}\`:\n\nTITLE: ${f.title}\nCLAIM: ${f.explanation}\n\nTry HARD to refute it. Inspect the actual code (\`git diff ${BASE}...HEAD\` and the files). Default to refuted=true if you are not confident the problem is real.`,
          { label: `verify:${d.key}#${i + 1}`, phase: 'Verify', schema: VERDICT }
        )
      )).then(votes => {
        const live = votes.filter(Boolean)
        // If every skeptic died, the finding is UNVERIFIED — never fail-open into "confirmed".
        const unverified = live.length === 0
        const kills = live.filter(v => v.refuted).length
        return { ...f, dimension: d.key, survived: !unverified && kills < NEEDED, unverified, votes: live }
      })
    )
  ).then(findings => ({ dimension: d.key, failed: res.review === null, findings }))
)

// ---- coverage accounting: a failed dimension must be visible, never silently dropped ----
const perDim = reviewed.filter(Boolean)
const failedDims = perDim.filter(r => r.failed).map(r => r.dimension)
const all = perDim.flatMap(r => r.findings).filter(Boolean)
const confirmed = all.filter(f => f.survived)
const unverified = all.filter(f => f.unverified)
const refuted = all.filter(f => !f.survived && !f.unverified)

if (failedDims.length) log(`⚠ coverage gap — dimensions still failing after ${REVIEW_ATTEMPTS} attempts: ${failedDims.join(', ')}`)
log(`${DIMENSIONS.length - failedDims.length}/${DIMENSIONS.length} dimensions reviewed — ${all.length} findings: ${confirmed.length} confirmed, ${refuted.length} refuted, ${unverified.length} unverified`)

const rank = { critical: 0, high: 1, medium: 2, low: 3 }
return {
  confirmed: confirmed.sort((a, b) => rank[a.severity] - rank[b.severity]),
  unverified: unverified.map(f => ({ file: f.file, title: f.title, severity: f.severity })),
  refuted: refuted.map(f => ({ file: f.file, title: f.title })),
  coverage: { reviewed: DIMENSIONS.length - failedDims.length, total: DIMENSIONS.length, failedDimensions: failedDims },
}
