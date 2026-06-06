<script>
  import { railView } from "../../stores.js";
  import ChatsPane from "./ChatsPane.svelte";
  import SettingsPane from "./SettingsPane.svelte";
  import ProfilePane from "./ProfilePane.svelte";
  import StarredPane from "./StarredPane.svelte";
  import ArchivedPane from "./ArchivedPane.svelte";
  import PrivacyPane from "./PrivacyPane.svelte";
  import StatusPane from "./StatusPane.svelte";
  import ChannelsPane from "./ChannelsPane.svelte";
  import CommunitiesPane from "./CommunitiesPane.svelte";
  import ContactsPane from "./ContactsPane.svelte";
  import CallsPane from "./CallsPane.svelte";
  import StoragePane from "./StoragePane.svelte";
  import PlaceholderPane from "./PlaceholderPane.svelte";
</script>

<section class="sidebar">
  {#if $railView === "settings"}
    <SettingsPane />
  {:else if $railView === "profile"}
    <ProfilePane />
  {:else if $railView === "starred"}
    <StarredPane />
  {:else if $railView === "archived"}
    <ArchivedPane />
  {:else if $railView === "privacy"}
    <PrivacyPane />
  {:else if $railView === "status"}
    <StatusPane />
  {:else if $railView === "channels"}
    <ChannelsPane />
  {:else if $railView === "communities"}
    <CommunitiesPane />
  {:else if $railView === "contacts"}
    <ContactsPane />
  {:else if $railView === "calls"}
    <CallsPane />
  {:else if $railView === "storage"}
    <StoragePane />
  {:else}
    <ChatsPane />
  {/if}
</section>
