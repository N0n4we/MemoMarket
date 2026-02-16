<script setup lang="ts">
import { ref } from "vue";
import { useApp } from "./useApp";

const {
  packs, installedIds, currentView, selectedPack, searchQuery, filterTag,
  filteredPacks, allTags,
  settingsState, settingsContentVisible,
  settingsBtnRef, settingsTitleRef, settingsBtnRect, settingsTitleRect,
  editPack,
  deletePack, toggleInstall,
  startCreate, startEdit, addRule, removeRule, addTag, removeTag,
  saveCurrentPack, viewPack, goBack,
  exportForMemoChat, exportPack, importPack,
  openSettings, closeSettings,
  // Local / Remote toggle
  viewMode, loadingRemote,
  // Channels (each = a backend server)
  channels, selectedChannelId, selectedChannel,
  publishing, publishError, publishSuccess,
  newChannelUrl, newChannelToken, addingChannel, addChannelError,
  addChannel, removeChannel, updateChannelToken, registerOnChannel,
  publishPack, openPublish, openChannels,
} = useApp();

const tagInput = ref("");
const regUsername = ref("");
const regDisplayName = ref("");

function handleAddTag() {
  if (tagInput.value.trim()) {
    addTag(tagInput.value);
    tagInput.value = "";
  }
}

function handleRegister() {
  if (regUsername.value.trim() && selectedChannelId.value) {
    registerOnChannel(selectedChannelId.value, regUsername.value.trim(), regDisplayName.value.trim() || regUsername.value.trim());
  }
}
</script>

<template>
  <main class="app-container">
    <header class="app-header" data-tauri-drag-region>
      <div class="header-left">
        <h1 data-tauri-drag-region>MemoMarket</h1>
        <div class="mode-toggle">
          <span class="mode-label" :class="{ active: viewMode === 'local' }">Local</span>
          <button class="toggle-switch" :class="{ remote: viewMode === 'remote' }" @click="viewMode = viewMode === 'local' ? 'remote' : 'local'">
            <span class="toggle-knob"></span>
          </button>
          <span class="mode-label" :class="{ active: viewMode === 'remote' }">Remote</span>
          <span v-if="loadingRemote" class="loading-dot"></span>
        </div>
      </div>
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
        <p v-if="viewMode === 'remote' && channels.length === 0">No channels configured. Add a backend server first.</p>
        <p v-else-if="viewMode === 'remote' && loadingRemote">Loading remote packs...</p>
        <p v-else-if="packs.length === 0 && viewMode === 'local'">No rule packs yet. Create one or import from MemoChat!</p>
        <p v-else>No packs match your search.</p>
      </div>

      <div class="pack-grid">
        <div v-for="pack in filteredPacks" :key="pack.id" class="pack-card" @click="viewPack(pack)">
          <div class="pack-card-header">
            <span class="pack-name">{{ pack.name || 'Untitled' }}</span>
            <span v-if="viewMode === 'local'" class="install-badge" :class="{ installed: installedIds.includes(pack.id) }">
              {{ installedIds.includes(pack.id) ? '✓' : '' }}
            </span>
          </div>
          <p class="pack-desc">{{ pack.description || 'No description' }}</p>
          <div class="pack-meta">
            <span v-if="pack.author" class="pack-author">{{ pack.author }}</span>
            <span v-if="(pack as any)._channelName" class="pack-channel-badge">{{ (pack as any)._channelName }}</span>
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
          <button class="action-btn publish-btn" @click="openPublish(selectedPack!)">Publish</button>
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

    <!-- Publish View -->
    <div v-if="currentView === 'publish' && selectedPack" class="publish-view">
      <div class="detail-header">
        <button class="back-btn" @click="goBack">&larr; Back</button>
        <button class="action-btn" @click="openSettings">Manage Channels</button>
      </div>
      <div class="publish-content">
        <h2>Publish "{{ selectedPack.name }}"</h2>

        <!-- Channel Selection -->
        <div class="publish-section">
          <h3>Select Channel</h3>
          <div v-if="channels.length === 0" class="empty-channel">
            <p>No channels configured. Add a backend server in Settings first.</p>
            <button class="action-btn primary" @click="openSettings">Open Settings</button>
          </div>
          <div v-else class="channel-select">
            <button
              v-for="ch in channels" :key="ch.id"
              class="channel-pill"
              :class="{ active: selectedChannelId === ch.id }"
              @click="selectedChannelId = ch.id"
            >
              <span class="channel-pill-name">{{ ch.name }}</span>
              <span class="channel-pill-url">{{ ch.url }}</span>
            </button>
          </div>
        </div>

        <!-- Selected channel details -->
        <div v-if="selectedChannel" class="publish-section">
          <h3>{{ selectedChannel.name }}</h3>
          <p v-if="selectedChannel.description" class="channel-desc">{{ selectedChannel.description }}</p>
          <p class="channel-url">{{ selectedChannel.url }}</p>
          <div v-if="!selectedChannel.token" class="register-section">
            <p class="hint">No token for this channel. Register or paste a token:</p>
            <div class="form-group">
              <label>Token</label>
              <input type="password" :value="selectedChannel.token" @input="updateChannelToken(selectedChannel!.id, ($event.target as HTMLInputElement).value)" placeholder="Paste auth token..." />
            </div>
            <div class="settings-divider"></div>
            <p class="hint">Or register a new account:</p>
            <div class="form-row">
              <div class="form-group"><label>Username</label><input type="text" v-model="regUsername" placeholder="username" /></div>
              <div class="form-group"><label>Display Name</label><input type="text" v-model="regDisplayName" placeholder="Your Name" /></div>
            </div>
            <button class="action-btn primary" @click="handleRegister" :disabled="!regUsername.trim()">Register</button>
          </div>
          <div v-else class="token-status">
            <span class="token-ok">Authenticated</span>
          </div>
        </div>

        <!-- Publish Action -->
        <div class="publish-action">
          <div v-if="publishError" class="publish-msg error">{{ publishError }}</div>
          <div v-if="publishSuccess" class="publish-msg success">{{ publishSuccess }}</div>
          <button
            class="action-btn primary publish-go"
            @click="publishPack(selectedPack!)"
            :disabled="publishing || !selectedChannel || !selectedChannel?.token"
          >{{ publishing ? 'Publishing...' : 'Publish' }}</button>
        </div>
      </div>
    </div>

    <!-- Channels View (redirect to Settings) -->
    <div v-if="currentView === 'channels'" class="channels-view">
      <div class="detail-header">
        <button class="back-btn" @click="goBack">&larr; Back</button>
        <button class="action-btn primary" @click="openSettings">Open Settings</button>
      </div>
      <div class="channels-content">
        <div class="empty-state">
          <p>Channel management has moved to Settings.</p>
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

        <!-- Channels Configuration -->
        <div class="settings-section">
          <h3>Channels</h3>
          <p class="channels-hint">Each channel is a backend server. Add server URLs to browse and publish packs.</p>

          <!-- Add channel form -->
          <div class="new-channel-form">
            <div class="form-group">
              <label>Server URL</label>
              <input type="text" v-model="newChannelUrl" placeholder="http://localhost:8080" @keydown.enter="addChannel" />
            </div>
            <div class="form-group">
              <label>Auth Token (optional)</label>
              <input type="password" v-model="newChannelToken" placeholder="Token if you already have one" />
            </div>
            <div v-if="addChannelError" class="publish-msg error">{{ addChannelError }}</div>
            <button class="action-btn primary" @click="addChannel" :disabled="addingChannel || !newChannelUrl.trim()">
              {{ addingChannel ? 'Connecting...' : '+ Add Channel' }}
            </button>
          </div>

          <div class="settings-divider"></div>

          <!-- Channel list -->
          <div v-if="channels.length === 0" class="empty-state" style="padding: 12px 0;">
            <p>No channels yet. Add a server URL above.</p>
          </div>
          <div class="channel-list">
            <div v-for="ch in channels" :key="ch.id" class="channel-card">
              <div class="channel-card-header">
                <span class="channel-card-name">{{ ch.name }}</span>
                <button class="remove-btn" @click="removeChannel(ch.id)">&times;</button>
              </div>
              <p class="channel-card-url">{{ ch.url }}</p>
              <p v-if="ch.description" class="channel-card-desc">{{ ch.description }}</p>
              <div class="channel-card-token">
                <span v-if="ch.token" class="token-ok">Authenticated</span>
                <span v-else class="token-missing">No token</span>
              </div>
            </div>
          </div>
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
