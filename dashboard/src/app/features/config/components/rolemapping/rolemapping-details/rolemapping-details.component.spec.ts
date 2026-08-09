import {ComponentFixture, TestBed} from '@angular/core/testing';

import {ConfigObject, Kind, Role, RoleMapping} from '../../../../../shared/models';
import {RoleMappingDetailsComponent} from './rolemapping-details.component';

describe('RoleMappingDetailsComponent', () => {
  let fixture: ComponentFixture<RoleMappingDetailsComponent>;
  let component: RoleMappingDetailsComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({imports: [RoleMappingDetailsComponent]}).compileComponents();
    fixture = TestBed.createComponent(RoleMappingDetailsComponent);
    component = fixture.componentInstance;
  });

  it('defaults a new mapping to a required email identity and role', () => {
    fixture.componentRef.setInput('configObject', new ConfigObject({kind: Kind.ROLEMAPPING}));
    fixture.detectChanges();

    expect(component.identityType.value).toBe('email');
    expect(component.email.enabled).toBe(true);
    expect(component.group.disabled).toBe(true);
    expect(component.form.valid).toBe(false);

    component.email.setValue('curator@example.test');
    component.roleList.setValue([Role.CURATOR]);
    expect(component.form.valid).toBe(true);
  });

  it('derives a group identity while editing and always clears the inactive identity', () => {
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'mapping-1',
      kind: Kind.ROLEMAPPING,
      roleMapping: new RoleMapping({group: 'archivists', roleList: [Role.CURATOR]}),
    }));
    fixture.detectChanges();

    expect(component.identityType.value).toBe('group');
    expect(component.group.value).toBe('archivists');
    expect(component.email.disabled).toBe(true);

    component.identityType.setValue('email');
    component.email.setValue('operator@example.test');
    let saved: ConfigObject | undefined;
    component.update.subscribe(value => saved = value);
    component.onUpdate();

    expect(saved?.roleMapping.email).toBe('operator@example.test');
    expect(saved?.roleMapping.group).toBe('');
  });

  it('rejects malformed active identities', () => {
    fixture.componentRef.setInput('configObject', new ConfigObject({kind: Kind.ROLEMAPPING}));
    fixture.detectChanges();
    component.roleList.setValue([Role.ADMIN]);
    component.email.setValue('not-an-email');
    expect(component.form.invalid).toBe(true);

    component.identityType.setValue('group');
    expect(component.email.disabled).toBe(true);
    expect(component.group.enabled).toBe(true);
    expect(component.form.invalid).toBe(true);
  });
});
