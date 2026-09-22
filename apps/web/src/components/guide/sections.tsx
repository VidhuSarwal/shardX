export const GUIDE_SECTIONS: { id: string; title: string; body: JSX.Element }[] = [
  {
    id: 'what',
    title: 'What ShardX is',
    body: (
      <>
        <p>ShardX is a zero-knowledge object store. A Go API and a React client work together to take a file off your device, obfuscate it, cut it into shards and spread those shards across your own AWS account — S3 for the bytes, KMS for encryption, DynamoDB for the bookkeeping.</p>
        <p>The obfuscation happens before anything leaves your browser session, and the seed that can undo it lives only in a key file you download. ShardX never stores that seed, so a breach of the bucket or the database on its own yields nothing readable.</p>
      </>
    ),
  },
  {
    id: 'account',
    title: 'Creating an account',
    body: (
      <>
        <p>Sign up with an email address and a password of at least 6 characters. Accounts are backed by an AWS Cognito user pool, so authentication and session tokens are handled by Cognito rather than by ShardX itself.</p>
        <p>Once your account exists, sign in from the same form to reach your vault.</p>
      </>
    ),
  },
  {
    id: 'upload',
    title: 'Uploading a file',
    body: (
      <>
        <p>Drop a file onto the uploader or select one manually. ShardX opens an upload session and streams the file in <strong>5 MB chunks</strong>. You can pause, resume or cancel the upload at any point — a chunk that fails to send retries on its own with exponential backoff, so a flaky connection does not mean starting over.</p>
        <p>Before the shards are written, you choose a sharding strategy:</p>
        <div className="overflow-x-auto">
          <table>
            <thead>
              <tr><th>Strategy</th><th>What it does</th><th>Use when</th></tr>
            </thead>
            <tbody>
              <tr><td><code>balanced</code></td><td>Equal-size shards across all targets</td><td>default; unknown target sizes</td></tr>
              <tr><td><code>greedy</code></td><td>Fills the target with the most free space first</td><td>one target is much larger</td></tr>
              <tr><td><code>proportional</code></td><td>Shard sizes proportional to each target&apos;s free space</td><td>mixed capacities</td></tr>
              <tr><td><code>manual</code></td><td>You pass explicit shard sizes</td><td>reproducible layouts / testing</td></tr>
            </tbody>
          </table>
        </div>
      </>
    ),
  },
  {
    id: 'finalize',
    title: 'After you finalize',
    body: (
      <>
        <p>Finalizing an upload returns immediately — the server does not make you wait for shards to finish writing. Instead, the client polls <code>/api/files/upload/status/&#123;id&#125;</code> every 3 seconds and walks the status through:</p>
        <ul>
          <li><code>uploading</code> — chunks are still being received</li>
          <li><code>processing</code> — the file is being obfuscated and sharded</li>
          <li><code>complete</code> — every shard is written and verified</li>
          <li><code>failed</code> — something in the pipeline could not be completed</li>
        </ul>
        <p>You can navigate away during processing; the status will be there when you check back.</p>
      </>
    ),
  },
  {
    id: 'detail',
    title: 'File detail & health',
    body: (
      <>
        <p>Each file has a detail view driven by the Integrity Engine. Its health percentage is <code>verified shards / total shards</code>, and the shard map colours each shard by state:</p>
        <ul>
          <li><strong>Green</strong> — verified</li>
          <li><strong>Amber</strong> — pending</li>
          <li><strong>Red</strong> — corrupted</li>
          <li><strong>Dark red</strong> — missing</li>
        </ul>
        <p>Below the map is an audit timeline built from EventBridge events recorded against the file: <code>FILE_CREATED</code>, <code>SHARD_VERIFIED</code>, <code>SHARD_QUEUED_FOR_RETRY</code>, <code>SHARD_CORRUPTED</code>, <code>FILE_HEALTHY</code>, <code>UPLOAD_FAILED</code>.</p>
      </>
    ),
  },
  {
    id: 'keyfile',
    title: 'The key file',
    body: (
      <>
        <p>Once a file reaches <code>complete</code>, you can download <code>&lt;filename&gt;.2xpfm.key</code>. It holds the obfuscation seed and the chunk map for that file — the two things needed to reverse what ShardX did to your data.</p>
        <p>The key file is generated client-side and is downloadable only after processing finishes. If you lose it, the file is unrecoverable: the server never held the seed, so it has nothing to regenerate the key from. Store it somewhere durable, separate from the vault itself.</p>
      </>
    ),
  },
  {
    id: 'drive',
    title: 'Drive mode',
    body: (
      <>
        <p>If you would rather not run an AWS bucket, ShardX can shard files across Google Drive instead, using the same obfuscate-and-shard pipeline. Link a Google account from your Profile page — it opens an OAuth popup and adds the account as a shard target once you approve it.</p>
        <p>Free space across linked accounts is pooled: three linked accounts at 15 GB each give you 45 GB of zero-knowledge storage with no cloud bill.</p>
      </>
    ),
  },
  {
    id: 'reliability',
    title: 'Reliability & retries',
    body: (
      <>
        <p>A shard upload that fails is not simply dropped. It is queued to SQS, and a background worker (<code>cmd/shardworker</code>) picks it up and retries automatically. If a shard still fails after 5 receives, it moves to a dead-letter queue and trips a CloudWatch alarm so the failure gets attention.</p>
        <p>While this is in flight, that shard shows as <strong>pending</strong> on the file detail view — it means a retry is already underway, not that anything is lost.</p>
      </>
    ),
  },
  {
    id: 'faq',
    title: 'FAQ',
    body: (
      <>
        <p><strong>Can ShardX read my files?</strong><br />No. Files are obfuscated with a seed that only exists in your key file. The server stores shards of noise, not your data.</p>
        <p><strong>What if I lose the key file?</strong><br />The file becomes unrecoverable. The seed is never stored server-side, so there is no way to regenerate it.</p>
        <p><strong>Is the file downloadable yet?</strong><br />Reconstruction is on the roadmap and not available today. Right now you can download the key file once a file reaches <code>complete</code>.</p>
        <p><strong>Which strategy should I pick?</strong><br /><code>balanced</code> is the sane default. Reach for <code>greedy</code> or <code>proportional</code> when your storage targets have very different capacities, and <code>manual</code> when you need a reproducible, exact layout.</p>
        <p><strong>Where do my shards live?</strong><br />In S3, encrypted with SSE-KMS under a customer-managed key, with each shard&apos;s SHA-256, size, bucket and region recorded in DynamoDB.</p>
        <p><strong>Does it work without AWS?</strong><br />Yes — Drive mode shards across linked Google Drive accounts instead, using the same pipeline.</p>
      </>
    ),
  },
];
