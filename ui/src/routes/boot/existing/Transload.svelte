<script>
    import { structure } from '$lib/stores/data'
    import { transloadPier } from '$lib/stores/websocket'
    import { createEventDispatcher } from 'svelte'

    const dispatch = createEventDispatcher();
    let filePath = '';
    $: patp = ($structure?.upload?.patp) || null

    const handleImport = () => {
        const validPathRegex = /^\/(?:[^\/]+\/)*[^\/]+(?:\.tar|\.tar\.gz|\.tgz|\.zip)$/;
        if (!validPathRegex.test(filePath.trim())) {
            alert('Please enter a valid file path with a .tar, .tar.gz, .zip, or .tgz extension');
        return;
        } else {
            transloadPier(patp, path, remote, fix, selectedDrive); // figure out how to get these
            filePath = '';
        }
    };
</script>
  
<div class="file-importer">
    <input type="text" bind:value={filePath} placeholder="Enter path on local disk (.tar.gz, .zip, .tar, .tgz)" />
    <button on:click={handleImport}>Import</button>
</div>
  
<style>
.file-importer {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: center;
    margin-top: 20px;
}
.file-importer input {
    flex-grow: 1;
    padding: 8px;
    border-radius: 4px;
    border: 2px solid var(--bg-base);
    background: white;
}
.file-importer button {
    padding: 8px 16px;
    border-radius: 4px;
    border: none;
    background-color: var(--accent-color, #000);
    color: white;
    cursor: pointer;
}
.file-importer button:hover {
    background-color: var(--accent-hover, #333);
}
</style>
