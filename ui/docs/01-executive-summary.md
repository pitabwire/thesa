# 1. Executive Summary

## What is Thesa UI?

Thesa UI is a Flutter application that works as a **universal control panel** for businesses. Think of it like a Swiss Army knife for managing any kind of backend system — whether that system handles orders, customers, inventory, finances, or anything else.

The key difference between Thesa UI and a normal application is this: **Thesa UI does not know what it is managing**. It has no built-in knowledge of orders, customers, or any specific business concept. Instead, it asks a server called the **BFF** (Backend-For-Frontend) what to show, and then it shows exactly that.

## The Restaurant Menu Analogy

Imagine a restaurant where the waiter (Thesa UI) does not have a fixed menu. Every morning, the kitchen (the BFF server) hands the waiter a new menu describing:

- What dishes are available today
- What each dish contains
- Which customers are allowed to order which dishes
- What steps are needed to prepare special orders
- How the dining room should be arranged

The waiter does not cook. The waiter does not decide the menu. The waiter reads the menu and serves the customers accordingly.

If the kitchen decides to add sushi tomorrow, the waiter does not need training. The menu just describes sushi, and the waiter serves it.

This is exactly how Thesa UI works. The BFF server sends descriptions of screens, forms, tables, actions, and workflows. Thesa UI reads those descriptions and renders them into a working user interface.

## Why Build It This Way?

### Problem: Tightly Coupled Frontends

In most companies, every backend system gets its own custom frontend. The order system has an order UI. The inventory system has an inventory UI. The finance system has a finance UI. Each frontend:

- Costs time and money to build
- Needs its own team to maintain
- Has inconsistent look and feel
- Duplicates common functionality (login, tables, forms, navigation)
- Must be rewritten when requirements change

### Solution: One UI That Adapts

Thesa UI solves this by building the UI once and letting the server control what it displays. When a new backend service is added:

1. The BFF exposes the new service's screens, forms, and actions
2. Thesa UI automatically renders them
3. No frontend code changes are needed

This means:

- **One codebase** serves unlimited backend systems
- **Consistent UX** across all systems
- **New features** appear by changing server responses, not frontend code
- **Permissions** are always enforced by the server, not the UI

## What Does "Offline-First" Mean?

Thesa UI saves everything the server tells it into a local database on the user's device. This means:

1. **Instant startup**: When a user opens the app, it does not wait for the server. It immediately shows the last-known screens from local storage.
2. **Works without internet**: If the network is down, users can still see their dashboards, browse cached data, and review information.
3. **Background updates**: While the user works, the app quietly checks the server for changes and updates the local data. The user sees smooth transitions, not loading spinners.

## Where Does Thesa UI Run?

Thesa UI is built with Flutter, which means it runs on:

| Platform | Example |
|---|---|
| **Web** | Chrome, Firefox, Safari — accessed via a URL |
| **Desktop** | Windows, macOS, Linux — installed as a native app |
| **Tablet** | iPad, Android tablets — with touch-optimized layouts |
| **Phone** | iPhone, Android phones — with compact layouts |

The same codebase produces all of these. The UI automatically adjusts its layout based on screen size.

## Who Is This For?

Thesa UI is designed for:

- **Enterprise companies** that need internal admin tools
- **Operations teams** that manage orders, workflows, and approvals
- **Platform companies** that run multiple backend services
- **Any organization** that wants a single, consistent admin interface

It can serve as the frontend for:

- ERP systems (enterprise resource planning)
- Internal tooling and admin panels
- Operations dashboards
- Marketplace back-offices
- Financial management systems

## What Does the BFF Do?

The BFF (Backend-For-Frontend) is a server that sits between Thesa UI and the actual backend systems. It:

1. **Aggregates** multiple backend APIs into a single interface
2. **Describes** what the UI should show (screens, forms, tables, actions)
3. **Enforces** permissions (who can see what, who can do what)
4. **Defines** workflows (multi-step processes like approvals)
5. **Provides** schemas (data structure definitions for forms and tables)

The BFF is NOT part of Thesa UI. It is a separate server that Thesa UI talks to. This document describes the Thesa UI frontend architecture only.

## Key Principles

These are the fundamental rules that guide every design decision:

| Principle | What It Means |
|---|---|
| **Capability-driven** | The UI only shows what the server says is available |
| **Schema-driven** | Forms and tables are built from data descriptions, not handwritten code |
| **Permission-driven** | If the server says "not allowed," the button does not exist — it is not greyed out, it is absent |
| **Workflow-aware** | Multi-step processes are first-class citizens, not afterthoughts |
| **Cache-first** | Always show cached data immediately; refresh in the background |
| **Domain-agnostic** | The UI has zero knowledge of business domains unless explicitly added via a plugin |
