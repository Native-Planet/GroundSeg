import { writable } from 'svelte/store'

import { Sha256 } from '@aws-crypto/sha256-browser';
import { checkPatp } from './patp';
import { uploadManifest } from './websocket'
import { toBase64 } from './gs-crypto'

// This is the upload process of a pier
export const processFile = async file => {
  // We get the name of the file and check if
  // it is a legit @p
  // If ilegitimate, we end the process
  const patp = file.name.split('.')[0]
  const legit = await checkPatp(patp)
  if (!legit) {
    console.log("failed")
    file = undefined
    return
  }
  // Here we get the required information to keep
  // track of the progress
  const chunkSize = 25 * 1024 * 1024; // 25MB
  const totalChunks = Math.ceil(file.size / chunkSize);
  let currentChunk = 0;
  let metadata = false

  // Then, we open a new transaction to IndexedDB with 
  // the @p of the pier
  const dbReq = indexedDB.open(patp, 1);
  let db;

  // Object store does not exist. Nothing to load
  dbReq.onupgradeneeded = function(event) {
    db = event.target.result;
    // Create the chunks store
    db.createObjectStore('chunks', { autoIncrement: true });
    // Create the metadata store
    db.createObjectStore('metadata', { autoIncrement: true });
  };

  // Sucessful storing the chunk in IndexDB
  dbReq.onsuccess = function(event) {
    db = event.target.result;
    if (!metadata) {
      processMetadata()
    }
    processNextChunk();
  };

  // Error handling
  dbReq.onerror = function(event) {
    console.error("IndexedDB error", event.target.error);
  };

  const processNextChunk = async () => {
    console.log(currentChunk + " / " + totalChunks)
    if (currentChunk >= totalChunks) {
      console.log("Done processing file");
      return;
    }
    let blob = file.slice(chunkSize * currentChunk, chunkSize * (currentChunk + 1));
    const reader = new FileReader();
    reader.onload = async function(event) {
      let arrayBuffer = event.target.result;
      let sha = new Sha256()
      sha.update(arrayBuffer)
      const res = await sha.digest()
      const hash = toBase64(res)
      let transaction = db.transaction(['chunks'], 'readwrite');
      let store = transaction.objectStore('chunks');
      let request = store.add({
        chunk: currentChunk,
        hash: hash,
        data: arrayBuffer
      });
      request.onsuccess = function() {
        currentChunk++;
        processNextChunk();
      };
      request.onerror = function() {
        console.error("Error adding chunk to IndexedDB", request.error);
      };
    };
    reader.onerror = function() {
      console.error("Error reading chunk", reader.error);
    };
    reader.readAsArrayBuffer(blob);
  };

  const processMetadata = async () => {
    console.log("size:",file.size)
    let blob = file
    const reader = new FileReader();
    reader.onload = async function(event) {
      let arrayBuffer = event.target.result;
      let sha = new Sha256()
      sha.update(arrayBuffer)
      const res = await sha.digest()
      const hash = toBase64(res)
      let transaction = db.transaction(['metadata'], 'readwrite');
      let store = transaction.objectStore('metadata');
      let request = store.add({
        hash: hash,
        size: file.size,
        chunks: totalChunks
      });
      request.onsuccess = function() {
        metadata = true
        processNextChunk();
      };
      request.onerror = function() {
        console.error("Error adding metadata to IndexedDB", request.error);
      };
    };
    reader.onerror = function() {
      console.error("Error reading file", reader.error);
    };
    reader.readAsArrayBuffer(blob);
  };
}
