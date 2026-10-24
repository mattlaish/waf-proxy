# WAF console operating guide

Use the console to connect your sites to application servers, choose protection
settings, and review security events. The console keeps its existing English
language; labels describe the action or result you can expect.

## Find a setting

| Console tab | What you can do |
|---|---|
| Dashboard | Check traffic, site availability, and recent security events. |
| Sites | Set domain names, public IP addresses and ports, certificates, server groups, and protection policies. |
| Application Servers | Add application servers, organize server groups, and check availability. |
| Policies | Choose security rules, inspection sensitivity, size limits, exceptions, IP access rules, and traffic limits. |
| API Security | Review discovered endpoints, expected fields, sign-in requirements, usage patterns, record-access findings, and GraphQL policies. |
| Logs | Search security events and access requests. |
| Setup / AI | Configure AI analysis, device synchronization, notifications, log delivery, TLS services, and software updates. |
| System | Run health checks and review request diagnostics, hardware keys, and acceleration. |
| Users | Manage accounts, permissions, and account activity. |
| Site Map | Review visited pages, scan linked pages, and customize protection for a page or field. |

## Save and activate changes

**Save draft** stores your changes without activating them. **Apply** activates
the draft after its settings have been checked. Applying can also activate other
saved changes, so review the complete draft first. The fixed action bar shows
whether settings are active, saved as a draft, or have unsaved edits.

Application server, AI, synchronization, notification, and log-delivery editors
save drafts. Use **Review changes** from any tab, or **Sites → Apply**, to see
the full change list and affected sites before activation. Connection tests use the
active settings, so apply the draft before testing new connection details.

Controls labeled **Save & apply**, **Apply traffic controls**, or **Apply**
activate their settings. API review, page protection, and account-management
actions take effect through their own controls. Reload rules reloads the active
rules; it does not activate an unrelated draft.

When leaving a tab with unsaved settings, choose **Save draft and continue**,
**Discard unsaved edits**, or **Continue editing**. Discard returns to the saved
draft, not necessarily the active settings. **Load active settings** replaces the
displayed settings only after confirmation; it does not delete the saved draft.
**Continue saved draft** opens the saved draft without activating it. Both actions
only read settings. **Discard saved draft** permanently removes the saved draft
after confirmation and loads active settings without changing live protection.
Only Operators and Administrators can discard it. The draft is shared on this
device, so this can remove someone else's saved changes too. There is no undo for
this deletion, and reloading will not bring the discarded draft back. Copy any
wanted edits first. If someone changed active settings while you were editing,
reload/reconcile instead of applying stale work.

Apply previews hide secrets, including secrets inside newly added objects.
Settings remain in browser memory until explicitly saved; they are not stored as
a local browser draft. Closing the browser can lose unsaved work even when the
browser offers a warning. Failed updates are not automatically repeated: check
the current result before trying again.

## Choose a protection mode

| Mode | What happens |
|---|---|
| Off | Standard security-rule inspection is off for the selected scope. |
| Monitor | Rules record matches without blocking requests through those rules. |
| Block | Security rules can reject requests according to the configured policy. |
| Learning | Supported API features collect information to build expected request patterns. |

API field and GraphQL policies start in **Learning**, then move to **Monitor**,
then **Block** when you choose. API sign-in policies start in **Monitor**.
Request-order monitoring and possible unauthorized record-access findings do not
offer blocking. AI suggestions do not enable API blocking.

These modes apply to the selected feature. Explicit page blocks, IP access rules,
traffic limits, and other separately enabled protections can still reject a request.

## Connect a site to your application

For guided setup, choose **Dashboard → Set up a site** or **Sites → Guided setup**.
The steps cover domain/listen address, server group, HTTPS/network and protection,
then show a summary. Save creates a draft only. Start in **Monitor**, apply after
review, then check server health and send an actual request through the WAF.
Configuration readiness and a TCP health check are not proof that the application
works. Hardware keys and external TLS services use the full site editor.
**Back** retains unfinished entries, including entries that still need correction;
validation happens when you move forward.

For manual setup:

1. In **Application Servers**, add each server’s IP address or hostname.
2. Create a server group. Add its servers, ports, traffic shares, and health checks.
3. In **Policies**, choose security rules and inspection sensitivity.
4. In **Sites**, enter the domain names, site IP and port, server group, and policy.
5. For HTTPS, select a certificate and private key, or configure a hardware key.
6. Save the draft, review your changes, and choose **Apply**.

To add a site IP automatically, enable **Add this site’s IP to a network
interface** and select an interface for application traffic. The management
interface cannot be selected. A network prefix of 0 uses the interface’s existing
prefix; enter one explicitly if the interface has no IP address.

## Set inspection sensitivity

**1 — Standard** uses the lowest sensitivity. Levels **2 — Increased**,
**3 — Strict**, and **4 — Maximum** check for more patterns and may require more
exceptions for legitimate requests. Inspect recorded requests before increasing
sensitivity or adding an exception.

Use the site’s policy to set header-inspection sensitivity. A page-specific
sensitivity setting affects request-content checks.

## Discover pages and form fields

**Live traffic** means a page or field was observed in a real request.
**Page scan** means it was found by **Scan linked pages**. The scan follows site
links and form submission addresses; it does not sign in, run JavaScript, submit
forms, or search for hidden pages.

For a login page or an application that loads forms with JavaScript, submit a
normal request through the WAF. When **Learn form fields from live requests** is
enabled, submitted field names can appear without a page scan. Field discovery
saves names rather than submitted values or uploaded files.

Scan progress updates automatically. Completion reports pages, forms and field
observations, with **Review discovered fields** to open their policies. Field
review is limited to 200 known page paths per comparison, with four parallel
reads; partial results are labelled. If a scan was already running when the
console opened, there may be no pre-scan baseline: the list is discovered fields,
not necessarily new fields. An open page editor is not rebuilt by scan polling.
**Clear** asks for confirmation and removes discovery/learning observations, not
applied protection policies.

## Customize a field

Open **Site Map**, select the site, and choose the settings button beside a page.
Review its form submission address and request methods, then add or edit a field.
Choose its type, allowed format, required setting, and length limits.
If a form submits to a different URL, select its address and press **Configure
this form**, even when there is only one address. Discovered field suggestions
follow you to that address. Suggestions are not enabled rules: use **Add suggested
rule** for each field you want to configure. Saving page notes or preparing an
exception for one field does not silently enable validation for other fields.

For example, a username can use **Username or identifier** while its password
uses **Password**. If a rule incorrectly flags a legitimate password, an exception
can apply only to that field and rule. Other fields remain inspected.

You can also use **Logs → event Details → Prepare a field-only exception**.
Choose the exact field and request method: older events do not record an
authoritative matched field or method, and the console does not guess.
The exception is prepared for review, not applied automatically. The active
policy summary explains the base policy and overlapping page settings in order.
It is not a simulation of every security rule.

**This page only** matches the exact URL. **URL starts with this path** is a text
prefix: /login also matches /login2. Field-validation methods control field format,
length and required checks; existing rule exceptions apply to the matching URL
for all methods. Page-wide rule exceptions affect every field and are broader
than the field-only exceptions.

Page changes update active protection directly, so apply or discard a pending
full configuration draft first. Review the page change before applying. The
editor preserves existing target-specific exceptions even if they are not shown
as editable fields. If exact/prefix rules share a URL, choose which one to edit.
Changing an existing policy's URL/scope requires an explicit remove/recreate
instead of silently leaving the old rule behind.

**Undo last page change** restores that policy's prior state from this browser
session. It is not a permanent version history and refuses known intervening
changes. Closing the browser loses this undo history. Page-edit close/Escape asks
before discarding unsaved work.
Undo remains available after deleting a prefix policy, even if the reopened editor
defaults to **This page only**. If both scopes have undo history, choose the policy
to restore. Undo refuses a known intervening change instead of overwriting it.

The **Allowed format** box accepts a validated regular expression for the whole
value. Double quotes and line breaks are not accepted in the expression itself.
Whether submitted values may contain quotes depends on the selected rule and
expression. The application must still handle submitted data safely.

**Review protection suggestions** proposes page-level rule exceptions and
sensitivity changes. It does not automatically produce field-specific exceptions.
**Page protection presets** offers settings for forms, URL parameters, JSON APIs,
uploads, and static files. **Apply matching suggestions** applies the displayed
confidence threshold only when you press it; the upload preset is not antivirus.

## Review API findings

Suggested field definitions need your approval before they can become a policy.
Use **Monitor** to check whether a proposed policy would affect legitimate traffic.
Move to **Block** only when ready.

Each API section loads separately. A slow or failed section does not hide results
from other sections. Errors remain visible with the last successful update, stale
result warnings, and **Retry** for reads. Permission errors explain that the
account lacks access. An unavailable/empty-looking table is not evidence that
no records exist. Use the section jump list and searchable site/method/URL
selectors rather than copying endpoint IDs.

**API usage patterns** shows unusual request order or skipped steps. **Possible
unauthorized record access** shows activity that may need investigation. Both
record findings without blocking requests. Record-access findings do not prove
ownership or replace application permission checks.

GraphQL policies can limit depth, complexity, fields, schema browsing, and data
changes. Field paths such as `query.viewer.orders.id` are input examples you may
need to enter exactly. Endpoint IDs, certificate paths, and protocol names remain
visible wherever they are needed to select or configure the correct resource.

## Check problems and updates

**Dashboard → Needs attention** links to unavailable application servers, recent
server-error responses, security matches, field suggestions, pending settings/
restart needs and certificates approaching expiry. Recent samples are not a
time-based attack rate: a security match need not be a blocked request, and
HTTP 403/429 can come from the application. Missing permission or failed reads
mean that some checks are unavailable, not that the system is healthy.

Certificate dates describe certificates held by active built-in Go listeners.
If an external TLS service terminates connections, or expiry is unavailable,
the console says so; verify that service's actual certificate separately.

Dialogs support Tab/Shift+Tab, Escape and focus return. New forms and page
settings show validation errors without activating incomplete settings.

In **System**, health checks describe the component and result. **Not checked**
means a check has not run; it does not mean the check passed. Request diagnostics
can record a selected site temporarily. Reviewers and Administrators can export
the results while protection remains active.

For software updates, verify the package and review its version before installing.
Use the operating system’s package manager for DEB/RPM installations. Standalone
installations may allow installation and rollback from the console when configured.

<!-- documentation-review: 2026-10-24; classification: operator-guide; current-authority: DOCUMENTATION_INDEX.md -->
