/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export type RegisteredJob = {
    /**
     * Unique identifier for the job
     */
    id: string;
    /**
     * Name of the job
     */
    job_name: string;
    /**
     * Human-readable description of the job
     */
    description?: string;
    /**
     * Cron schedule expression
     */
    schedule: string;
    /**
     * Whether this is a long-running job that needs heartbeats
     */
    is_long_running: boolean;
    /**
     * Whether the job is enabled
     */
    is_enabled: boolean;
    /**
     * When the job was last registered
     */
    last_registered_at: string;
    /**
     * ID of the instance that registered the job
     */
    instance_id: string;
    /**
     * Tenant ID the job belongs to
     */
    tenant_id: string;
    /**
     * When the job was first registered
     */
    created_at: string;
    /**
     * When the job was last updated
     */
    updated_at: string;
};

