import {ComponentFixture, TestBed} from '@angular/core/testing';

import {RoleMappingDialogComponent} from './rolemapping-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind, Role, RoleMapping} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('RoleMappingDialogComponent', () => {
  let component: RoleMappingDialogComponent;
  let fixture: ComponentFixture<RoleMappingDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.ROLEMAPPING}),
    options: {roles: [Role.ADMIN, Role.CURATOR, Role.READONLY]},
    allSelected: false
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [RoleMappingDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(RoleMappingDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders the active identity and role controls without collapsed flex fields', () => {
    const formFields = fixture.nativeElement.querySelectorAll('mat-form-field') as NodeListOf<HTMLElement>;
    expect(formFields).toHaveLength(3);
    expect(Array.from(formFields).every(field => !field.classList.contains('flex-fill'))).toBe(true);
    expect(fixture.nativeElement.querySelector('mat-select[formcontrolname="identityType"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('input[formcontrolname="email"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('input[formcontrolname="group"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-select[formcontrolname="roleList"]')).not.toBeNull();

    component.identityType.setValue('group');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('input[formcontrolname="email"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('input[formcontrolname="group"]')).not.toBeNull();
  });

  it('shows UPDATE for an existing mapping and enables it after a valid change', () => {
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'mapping-1',
      kind: Kind.ROLEMAPPING,
      roleMapping: new RoleMapping({group: 'curators', roleList: [Role.CURATOR]}),
    }));
    fixture.detectChanges();

    const updateButton = Array.from(
      fixture.nativeElement.querySelectorAll('mat-dialog-actions button') as NodeListOf<HTMLButtonElement>
    ).find(button => button.textContent.trim() === 'UPDATE');
    expect(updateButton).toBeDefined();
    expect(updateButton?.disabled).toBe(true);

    component.group.setValue('reviewers');
    component.group.markAsDirty();
    fixture.detectChanges();

    expect(updateButton?.disabled).toBe(false);
    const result = component.onDialogClose();
    expect(result.id).toBe('mapping-1');
    expect(result.roleMapping.group).toBe('reviewers');
    expect(result.roleMapping.roleList).toEqual([Role.CURATOR]);
  });

  it('uses the reactive identity selector and serializes only its active identity', () => {
    expect(component.identityType.value).toBe('email');
    component.email.setValue('user@example.test');
    component.roleList.setValue([Role.READONLY]);

    component.identityType.setValue('group');
    component.group.setValue('reviewers');
    const result = component.onDialogClose();

    expect(result.roleMapping.group).toBe('reviewers');
    expect(result.roleMapping.email).toBe('');
    expect(result.roleMapping.roleList).toEqual([Role.READONLY]);
  });
});
