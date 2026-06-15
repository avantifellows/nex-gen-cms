export const GRADE_COMMON_VALUE = "common";

export function isInt(value) {
    return Number.isInteger(Number(value));
}

export function isValidGradeFilter(value) {
    return isInt(value) || value === GRADE_COMMON_VALUE;
}