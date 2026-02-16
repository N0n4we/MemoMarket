<script setup lang="ts">
import { ref } from "vue";
import { useApp } from "./useApp";

const {
  packs, installedIds, currentView, selectedPack, searchQuery, filterTag,
  filteredPacks, allTags,
  apiKey, modelId, baseUrl, reasoningEnabled,
  settingsState, settingsContentVisible,
  settingsBtnRef, settingsTitleRef, settingsBtnRect, settingsTitleRect,
  editPack, generating,
  deletePack, toggleInstall,
  startCreate, startEdit, addRule, removeRule, addTag, removeTag,
  saveCurrentPack, viewPack, goBack,
  generateRules,
  exportForMemoChat, exportPack, importPack,
  openSettings, closeSettings,
} = useApp();

const tagInput = ref("");
const aiTopic = ref("");

function handleAddTag() {
  if (tagInput.value.trim()) {
    addTag(tagInput.value);
    tagInput.value = "";
  }
}

function handleGenerate() {
  if (aiTopic.value.trim()) {
    generateRules(aiTopic.value);
  }
}
</script>

<template>
  <main class="app-container">
    <header class="app-header" data-tauri-drag-region>
      <h1 data-tauri-drag-region>MemoMarket</h1>
      <div class="header-actions">
        <button class="action-btn" @click="importPack">Import</button>
        <button class="action-btn primary" @click="startCreate">+ New Pack</button>
        <button
          ref="settingsBtnRef"
          class="settings-btn"
          @click="openSettings"
          :class="{ 'settings-btn-hidden': settingsState !== 'closed' }"
        >Settings</button>
      </div>
    </header>

    <!-- Browse View -->
    <div v-if="currentView === 'browse'" class="browse-view">
      <div class="search-bar">
        <input type="text" v-model="searchQuery" placeholder="Search packs..." class="search-input" />
      </div>
      <div v-if="allTags.length > 0" class="tag-filter">
        <button class="tag-pill" :class="{ active: filterTag === '' }" @click="filterTag = ''">All</button>
        <button
          v-for="tag in allTags" :key="tag"
          class="tag-pill" :class="{ active: filterTag === tag }"
          @click="filterTag = filterTag === tag ? '' : tag"
        >{{ tag }}</button>
      </div>

      <div v-if="filteredPacks.length === 0" class="empty-state">
        <p v-if="packs.length === 0">No rule packs yet. Create one or import from MemoChat!</p>
        <p v-else>No packs match your search.</p>
      </div>

      <div class="pack-grid">
        <div v-for="pack in filteredPacks" :key="pack.id" class="pack-card" @click="viewPack(pack)">
          <div class="pack-card-header">
            <span class="pack-name">{{ pack.name || 'Untitled' }}</span>
            <span class="install-badge" :class="{ installed: installedIds.includes(pack.id) }">
              {{ installedIds.includes(pack.id) ? '✓' : '' }}
            </span>
          </div>
          <p class="pack-desc">{{ pack.description || 'No description' }}</p>
          <div class="pack-meta">
            <span v-if="pack.author" class="pack-author">{{ pack.author }}</span>
            <span class="pack-rules-count">{{ pack.rules.length }} rules</span>
          </div>
          <div v-if="pack.tags.length > 0" class="pack-tags">
            <span v-for="tag in pack.tags.slice(0, 3)" :key="tag" class="tag-small">{{ tag }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Detail View -->
    <div v-if="currentView === 'detail' && selectedPack" class="detail-view">
      <div class="detail-header">
        <button class="back-btn" @click="goBack">&larr; Back</button>
        <div class="detail-actions">
          <button class="action-btn" @click="exportForMemoChat(selectedPack!)">Export for MemoChat</button>
          <button class="action-btn" @click="exportPack(selectedPack!)">Export Pack</button>
          <button class="action-btn" @click="startEdit(selectedPack!)">Edit</button>
          <button
            class="action-btn" :class="{ primary: !installedIds.includes(selectedPack!.id) }"
            @click="toggleInstall(selectedPack!.id)"
          >{{ installedIds.includes(selectedPack!.id) ? 'Uninstall' : 'Install' }}</button>
          <button class="action-btn danger" @click="deletePack(selectedPack!.id)">Delete</button>
        </div>
      </div>
      <div class="detail-content">
        <h2>{{ selectedPack.name }}</h2>
        <p class="detail-desc">{{ selectedPack.description }}</p>
        <div class="detail-info">
          <span v-if="selectedPack.author">By {{ selectedPack.author }}</span>
          <span>v{{ selectedPack.version }}</span>
          <span>{{ selectedPack.rules.length }} rules</span>
        </div>
        <div v-if="selectedPack.tags.length > 0" class="detail-tags">
          <span v-for="tag in selectedPack.tags" :key="tag" class="tag-pill small">{{ tag }}</span>
        </div>
        <div v-if="selectedPack.systemPrompt" class="detail-section">
          <h3>System Prompt</h3>
          <div class="code-block">{{ selectedPack.systemPrompt }}</div>
        </div>
        <div class="detail-section">
          <h3>Rules</h3>
          <div v-for="(rule, idx) in selectedPack.rules" :key="idx" class="rule-card">
            <div class="rule-title">{{ rule.title || 'Untitled' }}</div>
            <div class="rule-update">{{ rule.updateRule }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit View -->
    <div v-if="currentView === 'create'" class="create-view">
      <div class="detail-header">
        <button class="back-btn" @click="goBack">&larr; Cancel</button>
        <button class="action-btn primary" @click="saveCurrentPack">Save Pack</button>
      </div>
      <div class="create-content">
        <div class="ai-section">
          <h3>AI Generate</h3>
          <div class="ai-row">
            <input type="text" v-model="aiTopic" placeholder="Describe what rules you need..." class="ai-input" @keydown.enter="handleGenerate" />
            <button class="action-btn primary" @click="handleGenerate" :disabled="generating || !aiTopic.trim()">
              {{ generating ? 'Generating...' : 'Generate' }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>Pack Name</label>
          <input type="text" v-model="editPack.name" placeholder="My Rule Pack" />
        </div>
        <div class="form-group">
          <label>Description</label>
          <input type="text" v-model="editPack.description" placeholder="What does this pack do?" />
        </div>
        <div class="form-row">
          <div class="form-group"><label>Author</label><input type="text" v-model="editPack.author" placeholder="Your name" /></div>
          <div class="form-group"><label>Version</label><input type="text" v-model="editPack.version" placeholder="1.0.0" /></div>
        </div>
        <div class="form-group">
          <label>System Prompt</label>
          <textarea v-model="editPack.systemPrompt" placeholder="System prompt for the AI..." rows="3"></textarea>
        </div>
        <div class="form-group">
          <label>Tags</label>
          <div class="tags-editor">
            <span v-for="(tag, idx) in editPack.tags" :key="idx" class="tag-pill small editable">
              {{ tag }}<button class="tag-remove" @click="removeTag(idx)">&times;</button>
            </span>
            <input type="text" v-model="tagInput" placeholder="Add tag..." class="tag-input" @keydown.enter.prevent="handleAddTag" />
          </div>
        </div>
        <div class="rules-section">
          <div class="section-header">
            <h3>Rules ({{ editPack.rules.length }})</h3>
            <button class="action-btn" @click="addRule">+ Add Rule</button>
          </div>
          <div v-for="(rule, idx) in editPack.rules" :key="idx" class="rule-edit-card">
            <div class="rule-edit-header">
              <span class="rule-num">#{{ idx + 1 }}</span>
              <button class="remove-btn" @click="removeRule(idx)">&times;</button>
            </div>
            <div class="form-group"><label>Title</label><input type="text" v-model="rule.title" placeholder="Rule title..." /></div>
            <div class="form-group"><label>Update Rule</label><input type="text" v-model="rule.updateRule" placeholder="How to update this memo..." /></div>
          </div>
          <button v-if="editPack.rules.length === 0" class="empty-add-btn" @click="addRule">+ Add your first rule</button>
        </div>
      </div>
    </div>

    <!-- Settings Panel -->
    <div
      v-if="settingsState !== 'closed'"
      class="settings-panel"
      :class="settingsState"
      :style="settingsState === 'expanding' || settingsState === 'collapsing' ? {
        '--btn-top': settingsBtnRect.top + 'px',
        '--btn-left': settingsBtnRect.left + 'px',
        '--btn-width': settingsBtnRect.width + 'px',
        '--btn-height': settingsBtnRect.height + 'px',
      } : {}"
    >
      <div class="settings-content" :class="{ 'content-visible': settingsContentVisible }">
        <div class="settings-header">
          <h2 ref="settingsTitleRef" :class="{ 'title-hidden': settingsState === 'expanding' || settingsState === 'collapsing' }">Settings</h2>
          <button class="close-btn" @click="closeSettings">&times;</button>
        </div>
        <div class="form-group">
          <label>Base URL</label>
          <input type="text" v-model="baseUrl" placeholder="https://api.openai.com/v1" />
        </div>
        <div class="form-group">
          <label>API Key</label>
          <input type="password" v-model="apiKey" placeholder="sk-..." />
        </div>
        <div class="form-group">
          <div class="label-row">
            <label>Model ID</label>
            <button type="button" class="reasoning-pill" :class="{ active: reasoningEnabled }" @click="reasoningEnabled = !reasoningEnabled">Reasoning</button>
          </div>
          <input type="text" v-model="modelId" placeholder="" />
        </div>
      </div>
    </div>

    <!-- Floating Settings text -->
    <div
      v-if="settingsState === 'expanding' || settingsState === 'collapsing'"
      class="floating-settings-text"
      :class="settingsState"
      :style="{
        '--btn-top': settingsBtnRect.top + 'px',
        '--btn-left': settingsBtnRect.left + 'px',
        '--btn-width': settingsBtnRect.width + 'px',
        '--btn-height': settingsBtnRect.height + 'px',
        '--title-top': settingsTitleRect.top + 'px',
        '--title-left': settingsTitleRect.left + 'px',
      }"
    >Settings</div>
  </main>
</template>
